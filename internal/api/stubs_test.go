package api

import (
	"context"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

// StubRepository implements EventRepository for testing.
type StubRepository struct {
	LoadCategorySummaryFn func(int) ([]*postgres.CategorySummary, error)
	LoadEventGroupsForCategoryFn func(string, int) ([]*postgres.EventGroup, error)
	SearchEventsFn        func(postgres.SearchQuery) ([]*postgres.EventGroup, error)
	LoadSimilarEventsFn   func(string, string) ([]*events.GenconEvent, error)
	LoadOrCreateUserFn    func(string) (*postgres.User, error)
	GetStarredIdsFn       func(string, int) (*postgres.UserStarredEvents, error)
	GetAllStarredIdsFn    func(string) (*postgres.UserStarredEvents, error)
	UpdateStarredEventFn  func(string, string, string, bool, bool) (*postgres.UserStarredEvents, error)
	UpdateStarredEventMinimalFn  func(string, string, string, bool, bool) (*postgres.UserStarredEvents, error)
	LoadStarredEventsFn   func(string, int) ([]*events.GenconEvent, error)
	LoadStarredEventGroupsFn func(string, int) ([]*postgres.EventGroup, error)
	LoadStarredEventClustersFn func(string, int, []*events.GenconEvent) ([]*postgres.CalendarEventCluster, error)
	ClearStarredEventsFn   func(string, int) error
	BulkStarEventsFn       func(string, int, []string, bool) error
	LoadAgendaFn           func(string, int) ([]*postgres.AgendaEntry, error)
	LoadPartiesFn          func(*postgres.User) ([]*postgres.Party, error)
	LoadPartyFn            func(int64) (*postgres.Party, error)
	NewPartyFn             func(string, int64, string) (*postgres.Party, error)
	UpdatePartyLeaderFn    func(int64, string) error
	RenamePartyFn          func(int64, string) error
	DeletePartyFn          func(int64) error
	RemoveMemberFn         func(int64, string) error
	JoinPartyFn            func(int64, string) error
	UpdateDisplayNameFn    func(string, string) error
	GetLastUpdateFn        func() (time.Time, error)
}

func (s *StubRepository) LoadCategorySummary(year int) ([]*postgres.CategorySummary, error) {
	return s.LoadCategorySummaryFn(year)
}
func (s *StubRepository) LoadEventGroupsForCategory(category string, year int) ([]*postgres.EventGroup, error) {
	return s.LoadEventGroupsForCategoryFn(category, year)
}
func (s *StubRepository) SearchEvents(q postgres.SearchQuery) ([]*postgres.EventGroup, error) {
	return s.SearchEventsFn(q)
}
func (s *StubRepository) LoadSimilarEvents(eventId string, userEmail string) ([]*events.GenconEvent, error) {
	return s.LoadSimilarEventsFn(eventId, userEmail)
}
func (s *StubRepository) LoadOrCreateUser(email string) (*postgres.User, error) {
	return s.LoadOrCreateUserFn(email)
}
func (s *StubRepository) GetStarredIds(email string, year int) (*postgres.UserStarredEvents, error) {
	return s.GetStarredIdsFn(email, year)
}
func (s *StubRepository) GetAllStarredIds(email string) (*postgres.UserStarredEvents, error) {
	return s.GetAllStarredIdsFn(email)
}
func (s *StubRepository) UpdateStarredEvent(email string, eventId string, tier string, starGroup bool, add bool) (*postgres.UserStarredEvents, error) {
	return s.UpdateStarredEventFn(email, eventId, tier, starGroup, add)
}
func (s *StubRepository) UpdateStarredEventMinimal(email string, eventId string, tier string, starGroup bool, add bool) (*postgres.UserStarredEvents, error) {
	return s.UpdateStarredEventMinimalFn(email, eventId, tier, starGroup, add)
}
func (s *StubRepository) LoadStarredEvents(email string, year int) ([]*events.GenconEvent, error) {
	return s.LoadStarredEventsFn(email, year)
}
func (s *StubRepository) LoadStarredEventGroups(email string, year int) ([]*postgres.EventGroup, error) {
	return s.LoadStarredEventGroupsFn(email, year)
}
func (s *StubRepository) LoadStarredEventClusters(email string, year int, starredEvents []*events.GenconEvent) ([]*postgres.CalendarEventCluster, error) {
	return s.LoadStarredEventClustersFn(email, year, starredEvents)
}
func (s *StubRepository) ClearStarredEvents(email string, year int) error {
	return s.ClearStarredEventsFn(email, year)
}
func (s *StubRepository) BulkStarEvents(email string, year int, eventIds []string, overwrite bool) error {
	return s.BulkStarEventsFn(email, year, eventIds, overwrite)
}
func (s *StubRepository) LoadAgenda(email string, year int) ([]*postgres.AgendaEntry, error) {
	return s.LoadAgendaFn(email, year)
}
func (s *StubRepository) LoadParties(user *postgres.User) ([]*postgres.Party, error) {
	return s.LoadPartiesFn(user)
}
func (s *StubRepository) LoadParty(id int64) (*postgres.Party, error) {
	if s.LoadPartyFn == nil {
		return nil, nil
	}
	return s.LoadPartyFn(id)
}
func (s *StubRepository) NewParty(name string, year int64, founderEmail string) (*postgres.Party, error) {
	return s.NewPartyFn(name, year, founderEmail)
}
func (s *StubRepository) UpdatePartyLeader(id int64, newLeaderEmail string) error {
	if s.UpdatePartyLeaderFn == nil {
		return nil
	}
	return s.UpdatePartyLeaderFn(id, newLeaderEmail)
}
func (s *StubRepository) RenameParty(id int64, name string) error {
	if s.RenamePartyFn == nil {
		return nil
	}
	return s.RenamePartyFn(id, name)
}
func (s *StubRepository) DeleteParty(id int64) error {
	if s.DeletePartyFn == nil {
		return nil
	}
	return s.DeletePartyFn(id)
}
func (s *StubRepository) RemoveMember(partyId int64, email string) error {
	if s.RemoveMemberFn == nil {
		return nil
	}
	return s.RemoveMemberFn(partyId, email)
}
func (s *StubRepository) JoinParty(partyId int64, email string) error {
	if s.JoinPartyFn == nil {
		return nil
	}
	return s.JoinPartyFn(partyId, email)
}
func (s *StubRepository) UpdateDisplayName(email string, name string) error {
	return s.UpdateDisplayNameFn(email, name)
}
func (s *StubRepository) GetLastUpdate() (time.Time, error) {
	if s.GetLastUpdateFn != nil {
		return s.GetLastUpdateFn()
	}
	return time.Time{}, nil
}

// StubAuthService implements AuthService for testing.
type StubAuthService struct {
	VerifyIDTokenFn func(context.Context, string) (string, error)
}

func (s *StubAuthService) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	return s.VerifyIDTokenFn(ctx, idToken)
}

// StubGameService implements GameService for testing.
type StubGameService struct {
	FindGameFn func(string) *postgres.Game
}

func (s *StubGameService) FindGame(name string) *postgres.Game {
	return s.FindGameFn(name)
}

// setupTestServer returns a configured Gin engine and stubs for testing.
func setupTestServer() (*Server, *StubRepository, *StubAuthService, *StubGameService, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	repo := &StubRepository{}
	auth := &StubAuthService{}
	games := &StubGameService{}

	server := NewServer(repo, auth, games)
	router := gin.New()

	return server, repo, auth, games, router
}
