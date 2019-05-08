package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tchap/gwi-challenge/cmd/gwi-api/api"
	"github.com/tchap/gwi-challenge/cmd/gwi-api/api/stores/memorystore"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAPI_Integration(t *testing.T) {
	assert := assert.New(t)

	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}

	// Prepare the store.
	store := memorystore.New()

	// Prepare the API object.
	a := api.New(zap.NewNop(), store, []byte("top-secret"))

	// Prepare Echo.
	e := echo.New()

	// Create a new account.
	c, _, rec := newRequest(t, e, "", bytes.NewBufferString(`{"email":"me@example.com","password":"secret"}`))
	c.SetPath("/v1/volunteers/login")

	must(a.PostLogin(c))

	// Get JWT token from the recorder.
	var body struct {
		Token string `json:"token"`
	}
	must(json.NewDecoder(rec.Body).Decode(&body))
	token := body.Token

	// Get me
	c, _, rec = newRequest(t, e, token, nil)
	c.SetPath("/v1/volunteers/me")

	must(a.GetMe(c))

	var me api.Volunteer
	must(json.NewDecoder(rec.Body).Decode(&me))
	assert.Equal(api.Volunteer{
		Email: "me@example.com",
	}, me)

	// Create a new team.
	c, _, rec = newRequest(t, e, token, bytes.NewBufferString(`{"id":"gophers","name":"The Gophers"}`))
	c.SetPath("/v1/teams")

	must(a.PostTeam(c))

	// Get the team.
	c, _, rec = newRequest(t, e, token, nil)
	c.SetPath("/v1/teams/:id")
	c.SetParamNames("id")
	c.SetParamValues("gophers")

	must(a.GetTeamByID(c))

	var team api.Team
	must(json.NewDecoder(rec.Body).Decode(&team))
	assert.Equal(api.Team{
		ID:   "gophers",
		Name: "The Gophers",
	}, team)

	// Join the team.
	c, _, rec = newRequest(t, e, token, nil)
	c.SetPath("/v1/teams/:id/members/:email")
	c.SetParamNames("id", "email")
	c.SetParamValues("gophers", "me@example.com")

	must(a.PutTeamMemberByEmail(c))

	// Get team members.
	c, _, rec = newRequest(t, e, token, nil)
	c.SetPath("/v1/teams/:id/members")
	c.SetParamNames("id")
	c.SetParamValues("gophers")

	must(a.GetTeamMembers(c))

	var members []api.Volunteer
	must(json.NewDecoder(rec.Body).Decode(&members))
	assert.Equal([]api.Volunteer{
		{
			Email: "me@example.com",
		},
	}, members)
}

func newRequest(
	t *testing.T,
	e *echo.Echo,
	token string,
	body io.Reader,
) (echo.Context, *http.Request, *httptest.ResponseRecorder) {

	req := httptest.NewRequest(http.MethodGet, "/", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if token != "" {
		var parser jwt.Parser
		jwtToken, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
		if err != nil {
			t.Fatal(err)
		}

		c.Set("user", jwtToken)
	}

	return c, req, rec
}
