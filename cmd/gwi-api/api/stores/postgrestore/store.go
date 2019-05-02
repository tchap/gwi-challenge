package postgrestore

import (
	"context"
	"database/sql"

	"github.com/tchap/gwi-challenge/cmd/gwi-api/api"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Store implements api.Store while keeping all data Postgres.
// pqcrypto must be enabled on the database being used.
type Store struct {
	logger *zap.Logger
	db     *sqlx.DB
}

// New returns a new Store instance using the given database.
func New(logger *zap.Logger, db *sqlx.DB) *Store {
	return &Store{
		logger: logger,
		db:     db,
	}
}

func (s *Store) AuthenticateOrCreateVolunteerAccount(ctx context.Context, volunteer *api.Volunteer) error {
	// Try to create a new account.
	// Check credentials on conflict.
	var authenticated bool
	if err := s.db.GetContext(ctx, &authenticated, `
		INSERT INTO volunteers (
			email,
			password
		)
		VALUES (
			$1,
			crypt($2, gen_salt('md5'))
		)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING volunteers.password = crypt($2, volunteers.password)
	`, volunteer.Email, volunteer.Password); err != nil {
		return s.queryError(err)
	}

	if !authenticated {
		return api.ErrPasswordMismatch
	}
	return nil
}

func (s *Store) GetVolunteerByEmail(ctx context.Context, email string) (*api.Volunteer, error) {
	var volunteer api.Volunteer
	if err := s.db.GetContext(ctx, &volunteer, `
		SELECT * FROM volunteers WHERE email = $1
	`, email); err != nil {
		return nil, s.queryError(err)
	}

	// We are actually only returning the email currently,
	// but this is a future-compatible solution.
	volunteer.Password = ""
	return &volunteer, nil
}

func (s *Store) CreateTeam(ctx context.Context, team *api.Team) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO teams (
			id,
			name
		) VALUES (
			$1,
			$2
		)
	`, team.ID, team.Name)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code.Name() == "unique_violation" {
				return api.ErrAlreadyExists
			}
		}
		return s.queryError(err)
	}
	return nil
}

func (s *Store) GetTeamByID(ctx context.Context, teamID string) (*api.Team, error) {
	var team api.Team
	if err := s.db.GetContext(ctx, &team, `
		SELECT * FROM teams WHERE id = $1
	`, teamID); err != nil {
		return nil, s.queryError(err)
	}
	return &team, nil
}

func (s *Store) AddTeamMember(ctx context.Context, teamID, email string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO team_members (
			team_id,
			volunteer_email
		) VALUES (
			$1,
			$2
		)
	`, teamID, email)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			switch err.Code.Name() {
			case "foreign_key_violation":
				return api.ErrNotFound
			case "unique_violation":
				return api.ErrAlreadyExists
			}
		}
		return s.queryError(err)
	}
	return nil
}

func (s *Store) ListTeamMembers(ctx context.Context, teamID string) ([]api.Volunteer, error) {
	// Get the rows.
	var volunteers []api.Volunteer
	err := s.db.SelectContext(ctx, &volunteers, `
		SELECT v.email
		FROM volunteers AS v INNER JOIN team_members AS tm ON v.email = tm.volunteer_email
		WHERE tm.team_id = $1
	`, teamID)
	if err != nil {
		return nil, s.queryError(err)
	}

	return volunteers, nil
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID, email string) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM team_members
		WHERE team_id = $1 AND volunteer_email = $2
	`, teamID, email)
	if err != nil {
		return s.queryError(err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return s.queryError(err)
	}
	if n == 0 {
		return api.ErrNotFound
	}
	return nil
}

func (s *Store) CountTeamMembers(ctx context.Context) (map[string]int, error) {
	// Get rows.
	rows, err := s.db.QueryContext(ctx, `
		SELECT team_id, COUNT(*) FROM team_members GROUP BY team_id 
	`)
	if err != nil {
		return nil, s.queryError(err)
	}

	// Go through rows and assemble the map.
	var (
		id    string
		count int

		counts = make(map[string]int)
	)
	for rows.Next() {
		if err := rows.Scan(&id, &count); err != nil {
			return nil, s.queryError(err)
		}

		counts[id] = count
	}
	if err := rows.Err(); err != nil {
		return nil, s.queryError(err)
	}

	return counts, nil
}

func (s *Store) RunHealthcheck(ctx context.Context) error {
	// Fail healthcheck when PING to the database fails.
	return s.db.PingContext(ctx)
}

func (s *Store) queryError(err error) error {
	if err == sql.ErrNoRows {
		return api.ErrNotFound
	}
	return errors.Wrap(err, "failed to execute database query")
}
