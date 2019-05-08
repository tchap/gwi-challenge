package main

import (
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// Config keeps all variables to be loaded from the environment.
type Config struct {
	// The following variables configure the main HTTP server.
	HTTPHost string `envconfig:"HTTP_HOST" default:"localhost"`
	HTTPPort int    `envconfig:"HTTP_PORT" default:"8888"`

	// The following options set the similarly named fields on http.Server instance.
	HTTPReadTimeout       time.Duration `envconfig:"HTTP_READ_TIMEOUT"`
	HTTPReadHeaderTimeout time.Duration `envconfig:"HTTP_READ_HEADER_TIMEOUT"`
	HTTPWriteTimeout      time.Duration `envconfig:"HTTP_WRITE_TIMEOUT"`
	HTTPIdleTimeout       time.Duration `envconfig:"HTTP_IDLE_TIMEOUT"`
	HTTPMaxHeaderBytes    int           `envconfig:"HTTP_MAX_HEADER_BYTES"`

	// Stats username/password is used to secure the stats API.
	StatsUsername string `envconfig:"STATS_USERNAME" required:"true"`
	StatsPassword string `envconfig:"STATS_PASSWORD" required:"true"`

	// JWTSecret is used to encode/decode JWT tokens.
	JWTSecret string `envconfig:"JWT_SECRET" required:"true"`

	// HealthcheckPort is used for the HTTP server processing healthcheck requests.
	HealthcheckPort int `envconfig:"HEALTHCHECK_PORT" default:"8899"`

	// DBDisabled can be set to false to use an in-memory store.
	DBURL      string `envconfig:"DB_URL"`
	DBDisabled bool   `envconfig:"DB_DISABLED"`

	// DebugEnabled can be set to enable debugging mode.
	DebugEnabled bool `envconfig:"DEBUG_ENABLED"`
}

// LoadConfig loads configuration from the environment.
func LoadConfig() (*Config, error) {
	var c Config
	if err := envconfig.Process("GWI_API", &c); err != nil {
		return nil, errors.Wrap(err, "failed to load configuration")
	}
	return &c, nil
}
