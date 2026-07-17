package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Encinarus/genconplanner/internal/postgres"
)

func TestAdminRoutes(t *testing.T) {
	server, stub, auth, _, r := setupTestServer()
	server.RegisterRoutes(r.Group("/api"))

	tests := []struct {
		name           string
		method         string
		url            string
		body           interface{}
		setupStub      func()
		expectedStatus int
	}{
		{
			name:   "Unauthorized - No Token",
			method: "GET",
			url:    "/api/v1/admin/orgs",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "", errors.New("invalid token")
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "Forbidden - Token Verified But Not Admin",
			method: "GET",
			url:    "/api/v1/admin/orgs",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "user@example.com", nil
				}
				stub.IsAdminFn = func(email string) (bool, error) {
					return false, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "Success - ViewOrgs",
			method: "GET",
			url:    "/api/v1/admin/orgs",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "admin@example.com", nil
				}
				stub.IsAdminFn = func(email string) (bool, error) {
					return true, nil
				}
				stub.LoadAllOrgsFn = func() ([]*postgres.Organizer, error) {
					return []*postgres.Organizer{
						{Id: 1, Aliases: []string{"Org A"}, NumEvents: 5},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "MergeOrgs - Success",
			method: "POST",
			url:    "/api/v1/admin/orgs/merge",
			body:   MergeOrgsRequest{Ids: []int64{1, 2}},
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "admin@example.com", nil
				}
				stub.IsAdminFn = func(email string) (bool, error) {
					return true, nil
				}
				stub.MergeOrgsFn = func(orgs []int64) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "MergeOrgs - BadRequest (Less than 2 IDs)",
			method: "POST",
			url:    "/api/v1/admin/orgs/merge",
			body:   MergeOrgsRequest{Ids: []int64{1}},
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "admin@example.com", nil
				}
				stub.IsAdminFn = func(email string) (bool, error) {
					return true, nil
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "MergeSuggestions - Success with similar items",
			method: "GET",
			url:    "/api/v1/admin/orgs/merge-suggestions",
			setupStub: func() {
				auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
					return "admin@example.com", nil
				}
				stub.IsAdminFn = func(email string) (bool, error) {
					return true, nil
				}
				stub.LoadAllOrgsFn = func() ([]*postgres.Organizer, error) {
					return []*postgres.Organizer{
						{Id: 1, Aliases: []string{"Fantasy Flight Games"}, NumEvents: 10},
						{Id: 2, Aliases: []string{"Fantasy Flight"}, NumEvents: 2},
					}, nil
				}
				stub.LoadEventOrgMetadataFn = func() ([]postgres.EventOrgMetadata, error) {
					return []postgres.EventOrgMetadata{
						{OrgGroup: "Fantasy Flight Games", Title: "Twilight Imperium Tournament", Year: 2026, GmNames: "Christian T. Petersen", Email: "contact@fantasyflight.com", Website: "fantasyflight.com"},
						{OrgGroup: "Fantasy Flight", Title: "Twilight Imperium Tournament", Year: 2026, GmNames: "Christian T. Petersen", Email: "contact@fantasyflight.com", Website: "fantasyflight.com"},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStub != nil {
				tt.setupStub()
			}

			var reqBody []byte
			if tt.body != nil {
				reqBody, _ = json.Marshal(tt.body)
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.url, bytes.NewBuffer(reqBody))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
