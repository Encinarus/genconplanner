package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Encinarus/genconplanner/internal/postgres"
)

func TestGetPartyByYear(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name         string
		param        string
		cookieValue  string
		setupStub    func()
		expectedCode int
	}{
		{
			name:        "Success - Load by Year",
			param:       "2026",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
					return &postgres.User{Email: email}, nil
				}
				stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
					return []*postgres.Party{
						{
							Id:          1,
							Name:        "2026 Party",
							Year:        2026,
							LeaderEmail: "test@example.com",
							ShortCode:   "CODE2026",
							Members: []*postgres.User{
								{Email: "test@example.com", DisplayName: "Test"},
							},
						},
					}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "Not Found - No Party for Year",
			param:       "2025",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
					return &postgres.User{Email: email}, nil
				}
				stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
					return []*postgres.Party{
						{
							Id:          1,
							Name:        "2026 Party",
							Year:        2026,
							LeaderEmail: "test@example.com",
							Members: []*postgres.User{
								{Email: "test@example.com", DisplayName: "Test"},
							},
						},
					}, nil
				}
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:        "Success - Load by Short Code",
			param:       "CODE123",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadPartyByCodeFn = func(code string) (*postgres.Party, error) {
					return &postgres.Party{
						Id:          2,
						Name:        "Code Party",
						Year:        2026,
						LeaderEmail: "test@example.com",
						ShortCode:   "CODE123",
						Members: []*postgres.User{
							{Email: "test@example.com", DisplayName: "Test"},
						},
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
			req, _ := http.NewRequest("GET", "/api/v1/party/"+tt.param, nil)
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

func TestCreatePartySinglePartyPerYear(t *testing.T) {
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
			name:        "Success - Create Party",
			cookieValue: "valid-token",
			body:        `{"name":"New Party","year":2026}`,
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
					return &postgres.User{Email: email}, nil
				}
				stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
					// User has no parties for 2026
					return []*postgres.Party{}, nil
				}
				stub.NewPartyFn = func(name string, year int64, email string) (*postgres.Party, error) {
					return &postgres.Party{
						Id:          1,
						Name:        name,
						Year:        year,
						LeaderEmail: email,
						ShortCode:   "NEWCODE",
						Members: []*postgres.User{
							{Email: email, DisplayName: "Test"},
						},
					}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "Rejected - Already in Party for Year",
			cookieValue: "valid-token",
			body:        `{"name":"Second Party","year":2026}`,
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
					return &postgres.User{Email: email}, nil
				}
				stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
					// User is already in a party for 2026
					return []*postgres.Party{
						{
							Id:          1,
							Name:        "Existing Party",
							Year:        2026,
							LeaderEmail: "test@example.com",
						},
					}, nil
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
			req, _ := http.NewRequest("POST", "/api/v1/user/parties", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
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

func TestJoinPartySinglePartyPerYear(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name         string
		param        string
		cookieValue  string
		setupStub    func()
		expectedCode int
	}{
		{
			name:        "Success - Join Party",
			param:       "JOINCODE",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadPartyByCodeFn = func(code string) (*postgres.Party, error) {
					return &postgres.Party{
						Id:          2,
						Name:        "Join Party",
						Year:        2026,
						LeaderEmail: "leader@example.com",
						ShortCode:   "JOINCODE",
					}, nil
				}
				stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
					return &postgres.User{Email: email}, nil
				}
				stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
					// User has no parties for 2026
					return []*postgres.Party{}, nil
				}
				stub.JoinPartyFn = func(partyId int64, email string) error {
					return nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "Rejected - Already in Party for Year",
			param:       "JOINCODE",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.LoadPartyByCodeFn = func(code string) (*postgres.Party, error) {
					return &postgres.Party{
						Id:          2,
						Name:        "Join Party",
						Year:        2026,
						LeaderEmail: "leader@example.com",
						ShortCode:   "JOINCODE",
					}, nil
				}
				stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
					return &postgres.User{Email: email}, nil
				}
				stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
					// User is already in a party for 2026
					return []*postgres.Party{
						{
							Id:          1,
							Name:        "Existing Party",
							Year:        2026,
							LeaderEmail: "test@example.com",
						},
					}, nil
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
			req, _ := http.NewRequest("POST", "/api/v1/party/"+tt.param+"/join", nil)
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
