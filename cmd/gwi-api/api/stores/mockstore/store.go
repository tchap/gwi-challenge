package mockstore

import (
	"context"

	"github.com/tchap/gwi-challenge/cmd/gwi-api/api"

	"github.com/stretchr/testify/mock"
)

// Store implements api.Store mock and can be used for testing.
type Store struct {
	mock.Mock
}

func (s *Store) AuthenticateOrCreateVolunteerAccount(ctx context.Context, volunteer *api.Volunteer) error {
	args := s.Called(ctx, volunteer)
	return args.Error(0)
}

func (s *Store) GetVolunteerByEmail(ctx context.Context, email string) (*api.Volunteer, error) {
	args := s.Called(ctx, email)
	v, err := args.Get(0), args.Error(1)
	if err != nil {
		return nil, err
	}
	return v.(*api.Volunteer), nil
}

func (s *Store) CreateTeam(ctx context.Context, team *api.Team) error {
	args := s.Called(ctx, team)
	return args.Error(0)
}

func (s *Store) GetTeamByID(ctx context.Context, teamID string) (*api.Team, error) {
	args := s.Called(ctx, teamID)
	v, err := args.Get(0), args.Error(1)
	if err != nil {
		return nil, err
	}
	return v.(*api.Team), nil
}

func (s *Store) AddTeamMember(ctx context.Context, teamID, email string) error {
	args := s.Called(ctx, teamID, email)
	return args.Error(0)
}

func (s *Store) ListTeamMembers(ctx context.Context, teamID string) ([]api.Volunteer, error) {
	args := s.Called(ctx, teamID)
	v, err := args.Get(0), args.Error(1)
	if err != nil {
		return nil, err
	}
	return v.([]api.Volunteer), nil
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID, email string) error {
	args := s.Called(ctx, teamID, email)
	return args.Error(0)
}

func (s *Store) CountTeamMembers(ctx context.Context) (map[string]int, error) {
	args := s.Called(ctx)
	v, err := args.Get(0), args.Error(1)
	if err != nil {
		return nil, err
	}
	return v.(map[string]int), nil
}

func (s *Store) RunHealthcheck(ctx context.Context) error {
	args := s.Called(ctx)
	return args.Error(0)
}
