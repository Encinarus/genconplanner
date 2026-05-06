package api

import (
	"context"

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
	UpdateStarredEventFn  func(string, string, bool, bool) (*postgres.UserStarredEvents, error)
	LoadStarredEventsFn   func(string, int) ([]*events.GenconEvent, error)
	LoadStarredEventGroupsFn func(string, int) ([]*postgres.EventGroup, error)
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
func (s *StubRepository) UpdateStarredEvent(email string, eventId string, starGroup bool, add bool) (*postgres.UserStarredEvents, error) {
	return s.UpdateStarredEventFn(email, eventId, starGroup, add)
}
func (s *StubRepository) LoadStarredEvents(email string, year int) ([]*events.GenconEvent, error) {
	return s.LoadStarredEventsFn(email, year)
}
func (s *StubRepository) LoadStarredEventGroups(email string, year int) ([]*postgres.EventGroup, error) {
	return s.LoadStarredEventGroupsFn(email, year)
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
