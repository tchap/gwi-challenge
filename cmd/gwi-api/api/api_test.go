package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tchap/gwi-challenge/cmd/gwi-api/api"
	"github.com/tchap/gwi-challenge/cmd/gwi-api/api/stores/mockstore"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// ----------------------------------------------------------------------------
// NOTE: Testing a single enpoint for time purposes only.
// ----------------------------------------------------------------------------

func TestAPI_GetTeamByID(t *testing.T) {
	assert := assert.New(t)

	// Testing endpoints returning *echo.HTTPError is kinda weird.
	// You cannot check the response object, you have to check the error object being returned.
	// So in case ExpectedResponseError is set, we do not check ExpectedResponseStatusCode at all.
	testCases := []struct {
		RequestedTeamID            string
		StoredTeam                 *api.Team
		ExpectedResponseError      error
		ExpectedResponseStatusCode int
		ExpectedResponseBody       echo.Map
	}{
		// Team found
		{
			RequestedTeamID: "gladiators",
			StoredTeam: &api.Team{
				ID:   "gladiators",
				Name: "The Gladiators",
			},
			ExpectedResponseStatusCode: http.StatusOK,
			ExpectedResponseBody: echo.Map{
				"id":   "gladiators",
				"name": "The Gladiators",
			},
		},
		// Team not found
		{
			RequestedTeamID:       "gladiators",
			ExpectedResponseError: echo.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		// Mock the store.
		store := &mockstore.Store{}
		if tc.StoredTeam != nil {
			store.
				On("GetTeamByID", mock.Anything, tc.RequestedTeamID).
				Return(tc.StoredTeam, nil)
		} else {
			store.
				On("GetTeamByID", mock.Anything, mock.Anything).
				Return(nil, api.ErrNotFound)
		}

		// Prepare the endpoint.
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetPath("/v1/teams/:id")
		c.SetParamNames("id")
		c.SetParamValues(tc.RequestedTeamID)

		a := api.New(zap.NewNop(), store, []byte("top-secret"))

		// Call the endpoint, at last.
		err := a.GetTeamByID(c)
		if tc.ExpectedResponseError != nil {
			assert.Equal(tc.ExpectedResponseError, err)
		} else {
			assert.NoError(err)
			assert.Equal(tc.ExpectedResponseStatusCode, rec.Code)

			if tc.ExpectedResponseBody != nil {
				body, _ := json.Marshal(tc.ExpectedResponseBody)
				assert.JSONEq(string(body), rec.Body.String())
			}
		}

	}
}
