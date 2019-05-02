package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

// DefaultTokenExpiration is the default JWT token expiration.
const DefaultTokenExpiration = 72 * time.Hour

var (
	// ErrPasswordMismatch is returned on password mismatch.
	ErrPasswordMismatch = errors.New("password mismatch")
	// ErrAlreadyExists is returned when a resource is to be created, but it exists already.
	ErrAlreadyExists = errors.New("already exists")
	// ErrNotFound is in general returned on resource not found.
	ErrNotFound = errors.New("not found")
)

// Volunteer represents the volunteer resource.
type Volunteer struct {
	Email    string `json:"email"              db:"email"`
	Password string `json:"password,omitempty" db:"password"`
}

// Team represents the team resource.
type Team struct {
	ID   string `json:"id"             db:"id"`
	Name string `json:"name,omitempty" db:"name"`
}

// Store is used by the API to store any data necessary.
type Store interface {
	// AuthenticateOrCreateVolunteerAccount tries to authenticate the given account.
	// If the account does not exist yet, it is automatically created.
	//
	// ErrPasswordMismatch must be returned when the account already exists,
	// yet the password provided does not match.
	AuthenticateOrCreateVolunteerAccount(ctx context.Context, volunteer *Volunteer) error
	// GetVolunteerByEmail finds a volunteer by their email address.
	// Password field MUST NOT be set.
	//
	// ErrNotFound is to be returned on volunteer not found.
	GetVolunteerByEmail(ctx context.Context, email string) (*Volunteer, error)

	// CreateTeam creates a new team.
	//
	// ErrAlreadyExists must be returned when a team with the given ID exists already.
	CreateTeam(ctx context.Context, team *Team) error
	// GetTeamByID finds a team resource by ID.
	//
	// ErrNotFound is to be returned on team not found.
	GetTeamByID(ctx context.Context, id string) (*Team, error)
	// AddTeamMember adds a new member to the given team.
	//
	// ErrNotFound is to be returned when the team or email is unknown.
	// ErrAlreadyExists is to be returned when already a member.
	AddTeamMember(ctx context.Context, teamID, volunteerEmail string) error
	// ListTeamMembers returns the members of the given team.
	// Password field MUST NOT be set.
	//
	// ErrNotFound is to be returned when the team is unknown.
	ListTeamMembers(ctx context.Context, teamID string) ([]Volunteer, error)
	// RemoveTeamMember removes a volunteer from the given team.
	//
	// ErrNotFound is to be returned when there is no such team or team member.
	RemoveTeamMember(ctx context.Context, teamID, volunteerEmail string) error

	// CountTeamMembers returns a map of team ID -> member count.
	CountTeamMembers(ctx context.Context) (map[string]int, error)

	// RunHealthcheck should return an error when the store is not healthy.
	RunHealthcheck(ctx context.Context) error
}

// Option can be used set an API option.
type Option func(*API)

// SetTokenExpiration sets the JWT token expiration.
func SetTokenExpiration(expiration time.Duration) Option {
	return func(api *API) {
		api.tokenExpiration = expiration
	}
}

// API contains all HTTP request handlers necessary.
type API struct {
	// TODO: logger is currently unused. Skipping this for time reasons,
	//       otherwise I would be logging API errors and some debug perhaps.
	logger      *zap.Logger
	store       Store
	tokenSecret []byte

	// Options
	tokenExpiration time.Duration
}

// New returns a new API instance.
func New(logger *zap.Logger, store Store, tokenSecret []byte, options ...Option) *API {
	api := &API{
		logger:          logger,
		store:           store,
		tokenSecret:     tokenSecret,
		tokenExpiration: DefaultTokenExpiration,
	}

	for _, opt := range options {
		opt(api)
	}

	return api
}

// PostLogin created a user session.
// It can be used to both sign up and log in.
//
// Returns:
//   * Bad Request on invalid request body
//   * Unauthorized on account exists but password mismatch
//   * OK on success, in which case "token" is present in the JSON response body
func (api *API) PostLogin(c echo.Context) error {
	// Decode the resource.
	var volunteer Volunteer
	if err := c.Bind(&volunteer); err != nil {
		return err
	}

	// Validate the body.
	switch {
	case volunteer.Email == "":
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	case volunteer.Password == "":
		return echo.NewHTTPError(http.StatusBadRequest, "password is required")
	}

	// Call the store.
	ctx := c.Request().Context()
	if err := api.store.AuthenticateOrCreateVolunteerAccount(ctx, &volunteer); err != nil {
		if err == ErrPasswordMismatch {
			return echo.ErrUnauthorized
		}
		return err
	}

	// Create a JWT token and return it.
	token, err := api.GenerateJWTToken(&volunteer)
	if err != nil {
		return err
	}

	return c.JSONPretty(http.StatusOK, echo.Map{
		"token": token,
	}, "  ")
}

// GetMe returns account details for the currently authenticated volunteer.
//
// Returns:
//   * OK on success
//
// NOTE: Not documenting other methods for time reasons.
func (api *API) GetMe(c echo.Context) error {
	// Get email for the current user.
	email, ok := api.GetEmailFromJWTToken(c)
	if !ok {
		return echo.ErrUnauthorized
	}

	// Load volunteer from the store.
	ctx := c.Request().Context()
	volunteer, err := api.store.GetVolunteerByEmail(ctx, email)
	if err != nil {
		return err
	}

	// Return the resource.
	return c.JSONPretty(http.StatusOK, volunteer, "  ")
}

func (api *API) PostTeam(c echo.Context) error {
	// Decode the resource.
	var team Team
	if err := c.Bind(&team); err != nil {
		return err
	}

	// Validate the body.
	switch {
	case team.ID == "":
		return echo.NewHTTPError(http.StatusBadRequest, "id is required")
	}

	// Call the store.
	ctx := c.Request().Context()
	if err := api.store.CreateTeam(ctx, &team); err != nil {
		if err == ErrAlreadyExists {
			return echo.NewHTTPError(http.StatusConflict)
		}
		return err
	}

	// Returning the resource created is the usual practice.
	return c.JSONPretty(http.StatusCreated, &team, "  ")
}

func (api *API) GetTeamByID(c echo.Context) error {
	// Load team from the store.
	teamID := c.Param("id")
	ctx := c.Request().Context()
	team, err := api.store.GetTeamByID(ctx, teamID)
	if err != nil {
		if err == ErrNotFound {
			return echo.ErrNotFound
		}
		return err
	}

	// Return the resource.
	return c.JSONPretty(http.StatusOK, team, "  ")
}

func (api *API) GetTeamMembers(c echo.Context) error {
	// Consult the store.
	teamID := c.Param("id")
	ctx := c.Request().Context()
	members, err := api.store.ListTeamMembers(ctx, teamID)
	if err != nil {
		return err
	}

	// Return team members. Make sure the slice is not nil to return [] on no member.
	if members == nil {
		members = []Volunteer{}
	}
	return c.JSONPretty(http.StatusOK, members, "  ")
}

func (api *API) PutTeamMemberByEmail(c echo.Context) error {
	teamID := c.Param("id")
	email := c.Param("email")
	ctx := c.Request().Context()
	if err := api.store.AddTeamMember(ctx, teamID, email); err != nil {
		switch err {
		case ErrNotFound:
			return echo.ErrNotFound
		case ErrAlreadyExists:
			return nil
		default:
			return err
		}
	}

	return c.NoContent(http.StatusCreated)
}

func (api *API) DeleteTeamMemberByEmail(c echo.Context) error {
	teamID := c.Param("id")
	email := c.Param("email")
	ctx := c.Request().Context()
	return api.store.RemoveTeamMember(ctx, teamID, email)
}

func (api *API) GetTeamMemberCounts(c echo.Context) error {
	ctx := c.Request().Context()

	// Count team members.
	counts, err := api.store.CountTeamMembers(ctx)
	if err != nil {
		return err
	}

	// Write response.
	return c.JSONPretty(http.StatusOK, counts, "  ")
}

func (api *API) GetHealthcheck(c echo.Context) error {
	return api.store.RunHealthcheck(c.Request().Context())
}

//
// Helpers
//

func (api *API) GenerateJWTToken(volunteer *Volunteer) (string, error) {
	// Init a new token.
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims.
	claims := token.Claims.(jwt.MapClaims)
	claims["email"] = volunteer.Email
	claims["exp"] = time.Now().Add(api.tokenExpiration).Unix()

	// Encode the token.
	return token.SignedString(api.tokenSecret)
}

func (api *API) GetEmailFromJWTToken(c echo.Context) (string, bool) {
	v := c.Get("user")
	if v == nil {
		return "", false
	}

	token, ok := v.(*jwt.Token)
	if !ok {
		return "", false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}

	email, ok := claims["email"].(string)
	return email, ok
}
