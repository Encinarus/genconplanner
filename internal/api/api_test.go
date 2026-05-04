package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

type StubRepository struct {
	LoadCategorySummaryFn func(int) ([]*postgres.CategorySummary, error)
	SearchEventsFn        func(postgres.SearchQuery) ([]*postgres.EventGroup, error)
	LoadSimilarEventsFn   func(string, string) ([]*events.GenconEvent, error)
	LoadOrCreateUserFn    func(string) (*postgres.User, error)
	GetStarredIdsFn       func(string) (*postgres.UserStarredEvents, error)
}

func (s *StubRepository) LoadCategorySummary(year int) ([]*postgres.CategorySummary, error) {
	return s.LoadCategorySummaryFn(year)
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
func (s *StubRepository) GetStarredIds(email string) (*postgres.UserStarredEvents, error) {
	return s.GetStarredIdsFn(email)
}

func TestCategoryValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &StubRepository{}
	r := gin.New()
	apiGroup := r.Group("/api/v1")
	categoryRoutes(apiGroup, stub)

	tests := []struct {
		name         string
		year         string
		setupStub    func()
		expectedCode int
	}{
		{
			name:         "Invalid year format",
			year:         "abc",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Year before 2020",
			year:         "2019",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Success path",
			year: "2024",
			setupStub: func() {
				stub.LoadCategorySummaryFn = func(year int) ([]*postgres.CategorySummary, error) {
					return []*postgres.CategorySummary{
						{Name: "Board Games", Code: "BGM", Count: 42},
					}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStub != nil {
				tt.setupStub()
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/category/"+tt.year, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}
