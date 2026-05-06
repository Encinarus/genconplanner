package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)


func TestCategoryValidation(t *testing.T) {
	server, stub, _, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

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
	server, stub, _, games, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

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
						{EventId: id, Title: "Test Event", ShortCategory: "BGM", GameSystem: "Catan"},
					}, nil
				}
				games.FindGameFn = func(name string) *postgres.Game {
					return &postgres.Game{Name: name}
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
			if tt.setupStub != nil {
				tt.setupStub()
			}
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
	server, stub, _, games, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

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
				stub.LoadEventGroupsForCategoryFn = func(cat string, year int) ([]*postgres.EventGroup, error) {
					return []*postgres.EventGroup{
						{Name: "Board Game Group", EventId: "BGM24123", Count: 1, GameSystem: "Catan"},
					}, nil
				}
				games.FindGameFn = func(name string) *postgres.Game {
					return &postgres.Game{Name: name}
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:  "Empty results",
			query: "?cat=NONE",
			setupStub: func() {
				stub.LoadEventGroupsForCategoryFn = func(cat string, year int) ([]*postgres.EventGroup, error) {
					return []*postgres.EventGroup{}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:  "Search by text",
			query: "?q=catan&year=2024",
			setupStub: func() {
				stub.SearchEventsFn = func(q postgres.SearchQuery) ([]*postgres.EventGroup, error) {
					if q.RawQuery == "catan" {
						return []*postgres.EventGroup{
							{Name: "Catan Event", EventId: "BGM24123", Count: 1, GameSystem: "Catan"},
						}, nil
					}
					return []*postgres.EventGroup{}, nil
				}
				games.FindGameFn = func(name string) *postgres.Game {
					return &postgres.Game{Name: name}
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
			req, _ := http.NewRequest("GET", "/api/v1/events"+tt.query, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	server, _, auth, _, r := setupTestServer()

	r.GET("/protected", server.AuthMiddleware(), func(c *gin.Context) {
		email := GetUserEmail(c)
		c.JSON(http.StatusOK, gin.H{"email": email})
	})

	tests := []struct {
		name         string
		cookieValue  string
		setupStub    func()
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Missing cookie",
			cookieValue:  "",
			expectedCode: http.StatusUnauthorized,
			expectedBody: `{"error":"Unauthorized"}`,
		},
		{
			name:        "Invalid token",
			cookieValue: "invalid",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "", fmt.Errorf("invalid token")
				}
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: `{"error":"Unauthorized"}`,
		},
		{
			name:        "Valid token",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
			},
			expectedCode: http.StatusOK,
			expectedBody: `{"email":"test@example.com"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStub != nil {
				tt.setupStub()
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/protected", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "signinToken", Value: tt.cookieValue})
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}
func TestLoadUserEvents(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name         string
		email        string
		year         string
		cookieValue  string
		setupStub    func()
		expectedCode int
	}{
		{
			name:         "Success path",
			email:        "test@example.com",
			year:         "2024",
			cookieValue:  "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.GetStarredIdsFn = func(email string, year int) (*postgres.UserStarredEvents, error) {
					return &postgres.UserStarredEvents{
						Email: email,
						StarredEvents: []postgres.StarredEvent{
							{EventId: "BGM24123", Level: "event"},
						},
					}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unauthorized - Email mismatch",
			email:        "other@example.com",
			year:         "2024",
			cookieValue:  "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Invalid year",
			email:        "test@example.com",
			year:         "abc",
			cookieValue:  "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStub != nil {
				tt.setupStub()
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/user/events/"+tt.email+"/"+tt.year, nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "signinToken", Value: tt.cookieValue})
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}
func TestGetUser(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name         string
		cookieValue  string
		setupStub    func()
		expectedCode int
		expectedBody string
	}{
		{
			name:        "Success path",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
					return &postgres.User{Email: email, DisplayName: "Test User"}, nil
				}
			},
			expectedCode: http.StatusOK,
			expectedBody: `{"email":"test@example.com","displayName":"Test User"}`,
		},
		{
			name:         "Unauthorized",
			cookieValue:  "",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:        "Service Unavailable",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
					return nil, fmt.Errorf("db error")
				}
			},
			expectedCode: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStub != nil {
				tt.setupStub()
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/user", nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "signinToken", Value: tt.cookieValue})
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestEventSearchPost(t *testing.T) {
	server, stub, _, games, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	stub.SearchEventsFn = func(q postgres.SearchQuery) ([]*postgres.EventGroup, error) {
		if q.CategoryShortCode == "BGM" && q.Year == 2024 {
			return []*postgres.EventGroup{{Name: "Board Game", EventId: "BGM123", Count: 1, GameSystem: "Catan"}}, nil
		}
		return []*postgres.EventGroup{}, nil
	}
	stub.LoadEventGroupsForCategoryFn = func(cat string, year int) ([]*postgres.EventGroup, error) {
		if cat == "BGM" && year == 2024 {
			return []*postgres.EventGroup{{Name: "Board Game", EventId: "BGM123", Count: 1, GameSystem: "Catan"}}, nil
		}
		return []*postgres.EventGroup{}, nil
	}
	games.FindGameFn = func(name string) *postgres.Game {
		return &postgres.Game{Name: name}
	}

	w := httptest.NewRecorder()
	// POST with form data
	req, _ := http.NewRequest("POST", "/api/v1/events", strings.NewReader("cat=BGM&year=2024"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestStarEvent(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name         string
		cookieValue  string
		body         string
		setupStub    func()
		expectedCode int
	}{
		{
			name:        "Success - Star one event",
			cookieValue: "valid-token",
			body:        `{"eventId":"BGM24123","add":true,"related":false}`,
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.UpdateStarredEventFn = func(email string, id string, related bool, add bool) (*postgres.UserStarredEvents, error) {
					return &postgres.UserStarredEvents{Email: email}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unauthorized - No token",
			cookieValue:  "",
			body:         `{"eventId":"BGM24123","add":true}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Bad request - Invalid JSON",
			cookieValue:  "valid-token",
			body:         `{"eventId":123}`, // EventId should be string
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStub != nil {
				tt.setupStub()
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/user/star", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "signinToken", Value: tt.cookieValue})
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

func TestGetStarredEvents(t *testing.T) {
	server, stub, auth, games, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name         string
		year         string
		cookieValue  string
		setupStub    func()
		expectedCode int
	}{
		{
			name:        "Success",
			year:        "2024",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadStarredEventGroupsFn = func(email string, year int) ([]*postgres.EventGroup, error) {
					return []*postgres.EventGroup{
						{EventId: "BGM24123", Name: "Starred Event", ShortCategory: "BGM", GameSystem: "Catan", Count: 1, ThursTickets: 10},
					}, nil
				}
				games.FindGameFn = func(name string) *postgres.Game {
					return &postgres.Game{Name: name}
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unauthorized",
			year:         "2024",
			cookieValue:  "",
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStub != nil {
				tt.setupStub()
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/user/starred/"+tt.year, nil)
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "signinToken", Value: tt.cookieValue})
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("%s: Expected status code %d, got %d", tt.name, tt.expectedCode, w.Code)
			}
		})
	}
}
