package memorystore

import (
	"context"
	"sync"

	"github.com/tchap/gwi-challenge/cmd/gwi-api/api"
)

// Store implements api.Store while keeping all data in memory only.
type Store struct {
	volunteers  map[string]*api.Volunteer
	teams       map[string]*api.Team
	teamMembers map[string]map[string]struct{}

	mu sync.RWMutex
}

func New() *Store {
	return &Store{
		volunteers:  make(map[string]*api.Volunteer),
		teams:       make(map[string]*api.Team),
		teamMembers: make(map[string]map[string]struct{}),
	}
}

func (s *Store) AuthenticateOrCreateVolunteerAccount(ctx context.Context, volunteer *api.Volunteer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.volunteers[volunteer.Email]
	if !ok {
		s.volunteers[volunteer.Email] = volunteer
		return nil
	}
	if v.Password != volunteer.Password {
		return api.ErrPasswordMismatch
	}
	return nil
}

func (s *Store) GetVolunteerByEmail(ctx context.Context, email string) (*api.Volunteer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.volunteers[email]
	if !ok {
		return nil, api.ErrNotFound
	}

	dupe := *v
	dupe.Password = ""
	return &dupe, nil
}

func (s *Store) CreateTeam(ctx context.Context, team *api.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.teams[team.ID]; ok {
		return api.ErrAlreadyExists
	}
	s.teams[team.ID] = team
	return nil
}

func (s *Store) GetTeamByID(ctx context.Context, teamID string) (*api.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.teams[teamID]
	if !ok {
		return nil, api.ErrNotFound
	}
	return t, nil
}

func (s *Store) AddTeamMember(ctx context.Context, teamID, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make sure the team exists.
	if _, ok := s.teams[teamID]; !ok {
		return api.ErrNotFound
	}

	// Make sure the volunteer exists.
	if _, ok := s.volunteers[email]; !ok {
		return api.ErrNotFound
	}

	members, ok := s.teamMembers[teamID]
	if ok {
		// Make sure the member is not there yet.
		if _, ok := members[email]; ok {
			return api.ErrAlreadyExists
		}
		members[email] = struct{}{}
	}
	s.teamMembers[teamID] = map[string]struct{}{
		email: struct{}{},
	}
	return nil
}

func (s *Store) ListTeamMembers(ctx context.Context, teamID string) ([]api.Volunteer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Make sure the team exists.
	if _, ok := s.teams[teamID]; !ok {
		return nil, api.ErrNotFound
	}

	emailSet := s.teamMembers[teamID]
	volunteers := make([]api.Volunteer, 0, len(emailSet))
	for email := range emailSet {
		// We may be dereferencing a nil pointer here,
		// but that would only happen on our data structures not being consistent.
		// Any team member must be registered as a volunteer.
		v := *s.volunteers[email]
		v.Password = ""
		volunteers = append(volunteers, v)
	}
	return volunteers, nil
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make sure the team exists.
	if _, ok := s.teams[teamID]; !ok {
		return api.ErrNotFound
	}

	members, ok := s.teamMembers[teamID]
	if !ok {
		return api.ErrNotFound
	}

	if _, ok := members[email]; !ok {
		return api.ErrNotFound
	}

	delete(members, email)
	return nil
}

func (s *Store) CountTeamMembers(ctx context.Context) (map[string]int, error) {
	counts := make(map[string]int, len(s.teamMembers))
	for k, v := range s.teamMembers {
		counts[k] = len(v)
	}
	return counts, nil
}

func (s *Store) RunHealthcheck(ctx context.Context) error {
	return nil
}
