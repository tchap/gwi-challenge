package postgrestore_test

import (
	"context"
	"testing"

	"github.com/tchap/gwi-challenge/cmd/gwi-api/api"
	"github.com/tchap/gwi-challenge/cmd/gwi-api/api/stores/postgrestore"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestStore_GetVolunteerByEmail(t *testing.T) {
	const (
		email    = "me@example.com"
		password = "<hash>"
	)

	// Mock the underlying DB.
	assert := assert.New(t)
	db, mock, err := sqlmock.New()
	assert.NoError(err)
	defer db.Close()

	// Init the store.
	store := postgrestore.New(zap.NewNop(), sqlx.NewDb(db, ""))

	// Mock the expected SQL statement.
	rows := sqlmock.NewRows([]string{"email", "password"}).
		AddRow(email, password)

	mock.
		ExpectQuery(`SELECT \* FROM volunteers WHERE email = \$1`).
		WithArgs(email).
		WillReturnRows(rows)

	// Call the store API.
	volunteer, err := store.GetVolunteerByEmail(context.Background(), email)
	assert.NoError(err)
	assert.Equal(&api.Volunteer{
		Email: email,
	}, volunteer)

	// Check mock DB expectations.
	assert.NoError(mock.ExpectationsWereMet())
}
