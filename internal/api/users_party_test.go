package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/Encinarus/genconplanner/internal/pubsub"
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
		expectedBody string
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
		{
			name:        "Success - Load by Short Code Non-Member Redaction",
			param:       "CODE123",
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "nonmember@example.com", nil
				}
				stub.LoadPartyByCodeFn = func(code string) (*postgres.Party, error) {
					return &postgres.Party{
						Id:          2,
						Name:        "Code Party",
						Year:        2026,
						LeaderEmail: "leader@example.com",
						ShortCode:   "CODE123",
						Members: []*postgres.User{
							{Email: "leader@example.com", DisplayName: "Leader", GenconName: "GCLeader", GenconId: "123", GenconEmail: "gcl@example.com"},
							{Email: "member@example.com", DisplayName: "Member", GenconName: "GCMember", GenconId: "456", GenconEmail: "gcm@example.com"},
						},
					}, nil
				}
			},
			expectedCode: http.StatusOK,
			expectedBody: `{"id":2,"name":"Code Party","year":2026,"leaderEmail":"leader@example.com","shortCode":"CODE123","inviteLink":"http://localhost:8080/party/CODE123","members":[{"displayName":"Leader","email":"leader@example.com","genconName":"","genconId":"","genconEmail":""},{"displayName":"Member","email":"","genconName":"","genconId":"","genconEmail":""}]}`,
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
				req.Header.Set("Authorization", "Bearer "+tt.cookieValue)
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("%s: Expected status code %d, got %d", tt.name, tt.expectedCode, w.Code)
			}
			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("%s: Expected body %s, got %s", tt.name, tt.expectedBody, w.Body.String())
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
				req.Header.Set("Authorization", "Bearer "+tt.cookieValue)
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
				req.Header.Set("Authorization", "Bearer "+tt.cookieValue)
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("%s: Expected status code %d, got %d", tt.name, tt.expectedCode, w.Code)
			}
		})
	}
}

func TestUpdatePartyMemberInfo(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name         string
		partyId      string
		body         string
		cookieValue  string
		setupStub    func()
		expectedCode int
	}{
		{
			name:        "Success - Leader updating member",
			partyId:     "100",
			body:        `{"email":"member@example.com","displayName":"New Member Name","genconName":"GCName","genconId":"123","genconEmail":"gencon@example.com"}`,
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "leader@example.com", nil
				}
				stub.LoadPartyFn = func(id int64) (*postgres.Party, error) {
					return &postgres.Party{
						Id:          100,
						Name:        "Test Party",
						Year:        2026,
						LeaderEmail: "leader@example.com",
						Members: []*postgres.User{
							{Email: "leader@example.com", DisplayName: "Leader"},
							{Email: "member@example.com", DisplayName: "Old Member"},
						},
					}, nil
				}
				stub.UpdatePartyMemberInfoFn = func(partyId int64, email, displayName, genconName, genconId, genconEmail string) error {
					if partyId != 100 || email != "member@example.com" || displayName != "New Member Name" || genconName != "GCName" || genconId != "123" || genconEmail != "gencon@example.com" {
						return fmt.Errorf("unexpected arguments")
					}
					return nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "Success - Member updating self",
			partyId:     "100",
			body:        `{"email":"member@example.com","displayName":"Self Name","genconName":"GCSelf","genconId":"456","genconEmail":"self@example.com"}`,
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "member@example.com", nil
				}
				stub.LoadPartyFn = func(id int64) (*postgres.Party, error) {
					return &postgres.Party{
						Id:          100,
						Name:        "Test Party",
						Year:        2026,
						LeaderEmail: "leader@example.com",
						Members: []*postgres.User{
							{Email: "leader@example.com", DisplayName: "Leader"},
							{Email: "member@example.com", DisplayName: "Old Member"},
						},
					}, nil
				}
				stub.UpdatePartyMemberInfoFn = func(partyId int64, email, displayName, genconName, genconId, genconEmail string) error {
					return nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "Forbidden - Member updating another member",
			partyId:     "100",
			body:        `{"email":"other@example.com","displayName":"Hacked Name","genconId":"789","genconEmail":"hacked@example.com"}`,
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "member@example.com", nil
				}
				stub.LoadPartyFn = func(id int64) (*postgres.Party, error) {
					return &postgres.Party{
						Id:          100,
						Name:        "Test Party",
						Year:        2026,
						LeaderEmail: "leader@example.com",
						Members: []*postgres.User{
							{Email: "leader@example.com", DisplayName: "Leader"},
							{Email: "member@example.com", DisplayName: "Member"},
							{Email: "other@example.com", DisplayName: "Other"},
						},
					}, nil
				}
			},
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStub != nil {
				tt.setupStub()
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/party/"+tt.partyId+"/member/update", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "signinToken", Value: tt.cookieValue})
				req.Header.Set("Authorization", "Bearer "+tt.cookieValue)
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("%s: Expected status code %d, got %d", tt.name, tt.expectedCode, w.Code)
			}
		})
	}
}

func TestRenameUserGenconInfo(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name         string
		body         string
		cookieValue  string
		setupStub    func()
		expectedCode int
	}{
		{
			name:        "Success - Update User Profile",
			body:        `{"displayName":"Updated User","genconName":"GCUser","genconId":"999","genconEmail":"user@gencon.com"}`,
			cookieValue: "valid-token",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "test@example.com", nil
				}
				stub.UpdateUserGenconInfoFn = func(email, displayName, genconName, genconId, genconEmail string) error {
					if email != "test@example.com" || displayName != "Updated User" || genconName != "GCUser" || genconId != "999" || genconEmail != "user@gencon.com" {
						return fmt.Errorf("unexpected arguments")
					}
					return nil
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
			req, _ := http.NewRequest("POST", "/api/v1/user/rename", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookieValue != "" {
				req.AddCookie(&http.Cookie{Name: "signinToken", Value: tt.cookieValue})
				req.Header.Set("Authorization", "Bearer "+tt.cookieValue)
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("%s: Expected status code %d, got %d", tt.name, tt.expectedCode, w.Code)
			}
		})
	}
}

type closeNotifierRecorder struct {
	*httptest.ResponseRecorder
	closeChan chan bool
}

func (r *closeNotifierRecorder) CloseNotify() <-chan bool {
	return r.closeChan
}

type errorResponseWriter struct {
	*closeNotifierRecorder
}

func (w *errorResponseWriter) Write(b []byte) (int, error) {
	return 0, fmt.Errorf("write error")
}

func TestPartyStream(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
		return "test@example.com", nil
	}
	stub.LoadPartyFn = func(id int64) (*postgres.Party, error) {
		if id != 123 {
			return nil, fmt.Errorf("party not found")
		}
		return &postgres.Party{
			Id:          123,
			Name:        "Stream Party",
			Year:        2026,
			LeaderEmail: "test@example.com",
			Members: []*postgres.User{
				{Email: "test@example.com", DisplayName: "Test"},
			},
		}, nil
	}

	t.Run("Streams interest updates and exits on context cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "GET", "/api/v1/party/123/stream", nil)
		req.AddCookie(&http.Cookie{Name: "signinToken", Value: "valid-token"})
		req.Header.Set("Authorization", "Bearer valid-token")

		w := &closeNotifierRecorder{
			ResponseRecorder: httptest.NewRecorder(),
			closeChan:        make(chan bool),
		}

		go func() {
			time.Sleep(50 * time.Millisecond)

			pubsub.Init()
			pubsub.PublishTestEvent(123, pubsub.PartyUpdateEvent{
				PartyId: 123,
				EventId: "BGM2612345",
				Email:   "test@example.com",
				Tier:    "must_have",
			})

			time.Sleep(50 * time.Millisecond)
			cancel()
			select {
			case w.closeChan <- true:
			default:
			}
		}()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "interest_update") || !strings.Contains(body, "BGM2612345") {
			t.Errorf("Expected stream to contain event data, got: %q", body)
		}
	})

	t.Run("Exits stream when write error occurs (c.IsAborted)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/party/123/stream", nil)
		req.AddCookie(&http.Cookie{Name: "signinToken", Value: "valid-token"})
		req.Header.Set("Authorization", "Bearer valid-token")

		w := &errorResponseWriter{
			closeNotifierRecorder: &closeNotifierRecorder{
				ResponseRecorder: httptest.NewRecorder(),
				closeChan:        make(chan bool),
			},
		}

		go func() {
			time.Sleep(50 * time.Millisecond)

			pubsub.Init()
			pubsub.PublishTestEvent(123, pubsub.PartyUpdateEvent{
				PartyId: 123,
				EventId: "BGM2612345",
				Email:   "test@example.com",
				Tier:    "must_have",
			})
		}()

		r.ServeHTTP(w, req)

		// Note: The HTTP status code might be 200 (since Gin set it initially),
		// but the handler must exit without getting stuck.
		// If the test finishes, it means the stream successfully terminated on the write error.
	})
}

func TestPartyAuthorizationHardening(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	t.Run("LeaveParty fails when caller is not a member", func(t *testing.T) {
		auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
			return "nonmember@example.com", nil
		}
		stub.LoadPartyFn = func(id int64) (*postgres.Party, error) {
			return &postgres.Party{
				Id:          123,
				Name:        "Target Party",
				Year:        2026,
				LeaderEmail: "leader@example.com",
				Members: []*postgres.User{
					{Email: "leader@example.com", DisplayName: "Leader"},
				},
			}, nil
		}
		
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/party/123/leave", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected leave party to return 404 for non-member, got %d", w.Code)
		}
	})

	t.Run("Querying party by numeric ID fails when caller is not a member", func(t *testing.T) {
		auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
			return "nonmember@example.com", nil
		}
		stub.LoadPartyFn = func(id int64) (*postgres.Party, error) {
			return &postgres.Party{
				Id:          123,
				Name:        "Secret Party",
				Year:        2026,
				LeaderEmail: "leader@example.com",
				Members: []*postgres.User{
					{Email: "leader@example.com", DisplayName: "Leader"},
				},
			}, nil
		}

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/party/123", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected querying party by ID to return 404 for non-member, got %d", w.Code)
		}
	})
}

