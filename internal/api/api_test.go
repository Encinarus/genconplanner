package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Encinarus/genconplanner/internal/background"
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

func TestEventLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &StubRepository{}
	// We need a dummy game cache
	db, _, _ := sqlmock.New()
	cache := background.NewGameCache(db)

	r := gin.New()
	apiGroup := r.Group("/api/v1")
	eventRoutes(apiGroup, stub, cache)

	tests := []struct {
		name         string
		eventId      string
		setupStub    func()
		expectedCode int
	}{
		{
			name:    "Success path",
			eventId: "BGM2412345",
			setupStub: func() {
				stub.LoadSimilarEventsFn = func(id string, email string) ([]*events.GenconEvent, error) {
					return []*events.GenconEvent{
						{EventId: id, Title: "Test Event", ShortCategory: "BGM"},
					}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:    "Not found",
			eventId: "MISSING123",
			setupStub: func() {
				stub.LoadSimilarEventsFn = func(id string, email string) ([]*events.GenconEvent, error) {
					return []*events.GenconEvent{}, nil
				}
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:    "Database error",
			eventId: "ERROR500",
			setupStub: func() {
				stub.LoadSimilarEventsFn = func(id string, email string) ([]*events.GenconEvent, error) {
					return nil, fmt.Errorf("db error")
				}
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupStub()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/event/"+tt.eventId, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func TestEventSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &StubRepository{}
	db, _, _ := sqlmock.New()
	cache := background.NewGameCache(db)

	r := gin.New()
	apiGroup := r.Group("/api/v1")
	eventRoutes(apiGroup, stub, cache)

	tests := []struct {
		name         string
		query        string
		setupStub    func()
		expectedCode int
	}{
		{
			name:  "Search by category",
			query: "?cat=BGM&year=2024",
			setupStub: func() {
				stub.SearchEventsFn = func(q postgres.SearchQuery) ([]*postgres.EventGroup, error) {
					return []*postgres.EventGroup{
						{Name: "Board Game Group", EventId: "BGM24123", Count: 1},
					}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:  "Empty results",
			query: "?cat=NONE",
			setupStub: func() {
				stub.SearchEventsFn = func(q postgres.SearchQuery) ([]*postgres.EventGroup, error) {
					return []*postgres.EventGroup{}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupStub()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/events"+tt.query, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}
