package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tchap/gwi-challenge/cmd/gwi-api/api"
	"github.com/tchap/gwi-challenge/cmd/gwi-api/api/stores/memorystore"
	"github.com/tchap/gwi-challenge/cmd/gwi-api/api/stores/postgrestore"
	"github.com/tchap/gwi-challenge/cmd/gwi-api/api/stores/postgrestore/migrations"

	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/tchap/zapext/types"
	"go.uber.org/zap"
	"gopkg.in/urfave/cli.v1"
)

// BuildVersion is set at compile time.
// In contains the current build version.
var BuildVersion = "UNSET"

func main() {
	app := cli.NewApp()
	app.Name = "gwi-api"
	app.Usage = "gwi-api service executable"
	app.Version = BuildVersion
	app.Action = run

	app.Commands = []cli.Command{
		{
			Name:  "db",
			Usage: "manage the database",
			Subcommands: []cli.Command{
				// TODO: Would be cool to add more subcommands,
				//       i.e. to migrate to particular migration step.
				{
					Name:   "migrate",
					Usage:  "migrate the database",
					Action: runDBMigrate,
				},
			},
		},
		{
			Name:    "healthcheck",
			Aliases: []string{"hc"},
			Usage:   "run service healthcheck",
			Action:  runHealthcheck,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error: %+v\n", err)
	}
}

func run(c *cli.Context) error {
	// Load configuration.
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Init logging.
	logger, err := initLogger(config.DebugEnabled)
	if err != nil {
		return err
	}
	defer logger.Sync()

	// TODO: Would be cool to add some Prometheus metrics.
	//       Leaving this as an exercise ;-)

	// XXX: Postgres store (including tests)
	// XXX: Documentation
	// XXX: Docker Compose

	// Init the store.
	store, closeStore, err := initStore(logger, config)
	if err != nil {
		return err
	}
	defer closeStore()

	// Init the API.
	a := api.New(logger, store, []byte(config.JWTSecret))

	// Run the healthcheck server in the background.
	if err := spawnHealthcheckServer(logger, config.HealthcheckPort, a.GetHealthcheck); err != nil {
		return err
	}

	// Run the API server.
	return runAPIServer(logger, config, a)
}

func initLogger(debugEnabled bool) (*zap.Logger, error) {
	if debugEnabled {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

func initStore(logger *zap.Logger, config *Config) (store api.Store, cleanup func(), err error) {
	if config.DBDisabled {
		return memorystore.New(), func() {}, nil
	}

	db, err := sqlx.Open("postgres", config.DBURL)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to open database")
	}

	return postgrestore.New(logger, db), func() { db.Close() }, nil
}

func spawnHealthcheckServer(logger *zap.Logger, port int, handler echo.HandlerFunc) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return errors.Wrapf(err, "failed to listen on port %d", port)
	}

	e := echo.New()
	e.GET("/", handler)

	logger.Info("healthcheck server starting...", zap.Int("port", port))

	go http.Serve(listener, e)
	return nil
}

func runAPIServer(logger *zap.Logger, config *Config, a *api.API) error {
	// Init Echo.
	e := initEcho(logger, config, a)

	// Init the HTTP server.
	server := initHTTPServer(config, e)

	// Keep processing requests until interrupted.
	logger.Info("API server starting...",
		zap.String("host", config.HTTPHost), zap.Int("port", config.HTTPPort))

	return runServerUntilInterrupted(logger, server)
}

func initHTTPServer(config *Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf("%v:%v", config.HTTPHost, config.HTTPPort),
		Handler:           handler,
		ReadTimeout:       config.HTTPReadTimeout,
		ReadHeaderTimeout: config.HTTPReadHeaderTimeout,
		WriteTimeout:      config.HTTPWriteTimeout,
		IdleTimeout:       config.HTTPIdleTimeout,
		MaxHeaderBytes:    config.HTTPMaxHeaderBytes,
	}
}

func initEcho(logger *zap.Logger, config *Config, a *api.API) *echo.Echo {
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())

	// Logging kinda sucks with Echo.
	// Not feeling like implementing echo.Logger interface,
	// it is not really compatible with zap.Logger approach.
	if config.DebugEnabled {
		// Log every request when debug is enabled.
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				req := c.Request()
				res := c.Response()
				reqID := res.Header().Get(echo.HeaderXRequestID)
				logger.Debug(
					"request received",
					zap.String("request_id", reqID),
					zap.Object("request", types.HTTPRequest{R: req}),
				)

				start := time.Now()
				err := next(c)
				delta := time.Since(start)

				code := res.Status
				if err != nil {
					if he, ok := err.(*echo.HTTPError); ok {
						code = he.Code
					} else {
						code = http.StatusInternalServerError
					}
				}

				logger.Debug(
					"response sent",
					zap.String("request_id", reqID),
					zap.Int("status_code", code),
					zap.Duration("duration", delta),
					zap.Error(err),
				)
				return err
			}
		})
	} else {
		// Otherwise just log errors.
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				err := next(c)
				if err != nil {
					req := c.Request()
					res := c.Response()
					logger.Error(
						"request failed",
						zap.Error(err),
						zap.String("request_id", res.Header().Get(echo.HeaderXRequestID)),
						zap.Object("request", types.HTTPRequest{R: req}),
					)
				}
				return err
			}
		})
	}

	jwtMiddleware := middleware.JWT([]byte(config.JWTSecret))

	limitEmailToCurrentUser := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Make sure the token email and param email match.
			email, ok := a.GetEmailFromJWTToken(c)
			emailParam := c.Param("email")
			if !ok || email != emailParam {
				return echo.ErrUnauthorized
			}
			// Call the next handler.
			return next(c)
		}
	}

	// Volunteers API
	v := e.Group("/v1/volunteers")
	v.POST("/login", a.PostLogin)
	v.GET("/me", a.GetMe, jwtMiddleware)

	// Teams API
	t := e.Group("/v1/teams", jwtMiddleware)
	t.POST("", a.PostTeam)
	t.GET("/:id", a.GetTeamByID)
	t.GET("/:id/members", a.GetTeamMembers)
	t.PUT("/:id/members/:email", a.PutTeamMemberByEmail, limitEmailToCurrentUser)
	t.DELETE("/:id/members/:email", a.DeleteTeamMemberByEmail, limitEmailToCurrentUser)

	// Stats API (uses Basic Auth)
	statsAuth := func(username, password string, c echo.Context) (bool, error) {
		return username == config.StatsUsername && password == config.StatsPassword, nil
	}

	s := e.Group("/v1/stats", middleware.BasicAuth(statsAuth))
	s.GET("/teams/member-count", a.GetTeamMemberCounts)

	return e
}

func runServerUntilInterrupted(logger *zap.Logger, server *http.Server) error {
	// Start catching signals. This works fine on Unix only.
	// Terminate the server gracefully on signal received.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	termAckCh := make(chan struct{})
	go func() {
		// Wait for a signal.
		sig := <-sigCh
		logger.Info("signal received", zap.String("signal", sig.String()))
		// Stop catching signals to die on the next one immediately.
		signal.Stop(sigCh)
		// Trigger indefinite shutdown.
		server.Shutdown(context.Background())
		// Notify the main thread we are done cleaning up.
		close(termAckCh)
	}()

	// Start the server.
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return errors.Wrap(err, "HTTP server crashed")
	}

	// Wait for the shutdown to finish and return.
	<-termAckCh
	return nil
}

func runDBMigrate(c *cli.Context) error {
	// Load config from the environment.
	// We only need the DB URL, but we are in the configured Docker image anyway, probably.
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Get the migrations source stored in a package.
	sourceDriver, err := migrations.NewSource()
	if err != nil {
		return err
	}

	// Init the migration using the configured DB URL.
	m, err := migrate.NewWithSourceInstance("go-bindata", sourceDriver, config.DBURL)
	if err != nil {
		return errors.Wrap(err, "failed to init database migration")
	}

	// Migrate all the way up!
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "failed to perform database migration")
	}
	return nil
}

func runHealthcheck(c *cli.Context) error {
	// Load config from the environment.
	// We need this to get the API address.
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Get the healthcheck endpoint.
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d", config.HealthcheckPort))
	if err != nil {
		return errors.Wrap(err, "failed to call the healthcheck endpoint")
	}
	// We are expecting 200 OK, otherwise we fail the healthcheck.
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code received: %d", resp.StatusCode)
	}

	fmt.Println("SUCCESS")
	return nil
}
