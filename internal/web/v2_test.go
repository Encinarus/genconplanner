package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

func TestServeV2(t *testing.T) {
	// 1. Setup mock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	// 2. Setup mock filesystem (create static/v2/index.html)
	// We use the current directory but we'll cleanup
	origDir, _ := os.Getwd()
	tmpDir, _ := os.MkdirTemp("", "v2test")
	defer os.RemoveAll(tmpDir)

	if errChdir := os.Chdir(tmpDir); errChdir != nil {
		t.Fatalf("failed to change directory to temp: %s", errChdir)
	}
	defer func() {
		if errChdir := os.Chdir(origDir); errChdir != nil {
			t.Logf("failed to restore directory: %s", errChdir)
		}
	}()

	err = os.MkdirAll("static/v2", 0755)
	if err != nil {
		t.Fatalf("failed to create static/v2: %s", err)
	}

	indexContent := `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>GenCon Planner</title><base href="/"></head><body><app-root></app-root></body></html>`
	err = os.WriteFile("static/v2/index.html", []byte(indexContent), 0644)
	if err != nil {
		t.Fatalf("failed to write index.html: %s", err)
	}

	err = os.MkdirAll("templates", 0755)
	if err != nil {
		t.Fatalf("failed to create templates: %s", err)
	}
	metaContent := `{{ define "meta" }}
<meta property="og:title" content="{{ .event.Title }}" />
<meta property="og:description" content="{{ .event.ShortDescription }}" />
<meta property="twitter:label1" content="Game" />
<meta property="twitter:data1" content="{{ .event.GameSystem }}" />
{{ end }}`
	err = os.WriteFile("templates/meta.html", []byte(metaContent), 0644)
	if err != nil {
		t.Fatalf("failed to write meta.html: %s", err)
	}

	// 3. Setup Gin
	gin.SetMode(gin.TestMode)

	// Create a cache (empty is fine for this test)
	cache := background.NewGameCache(db)

	t.Run("Event route with meta tag injection", func(t *testing.T) {
		// Mock the query performed by postgres.LoadSimilarEvents
		mock.ExpectQuery("SELECT distinct").
			WithArgs("BGM24123", 2024, "test@example.com").
			WillReturnRows(sqlmock.NewRows([]string{
				"event_id", "year", "active", "org_group", "title", "short_description",
				"long_description", "event_type", "game_system", "rules_edition",
				"min_players", "max_players", "age_required", "experience_required",
				"materials_provided", "start_time", "duration", "end_time", "gm_names",
				"website", "email", "tournament", "round_number", "total_rounds",
				"min_play_time", "attendee_registration", "cost", "location", "room_name",
				"table_number", "special_category", "tickets_available", "last_modified",
				"short_category", "is_starred", "org_id",
			}).AddRow(
				"BGM24123", 2024, true, "Org", "Test Event Title", "Short Desc",
				"Long Desc", "BGM", "Catan", "1st", 3, 4, "12+", "None",
				true, time.Now(), 60, time.Now(), "GM",
				"example.com", "gm@example.com", false, 1, 1, 60, "None", 5,
				"Room", "Room Name", "T1", "", 10, time.Now(), "BGM",
				false, 1,
			))

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("context", &Context{Email: "test@example.com"})
			c.Next()
		})
		r.NoRoute(ServeV2(db, cache))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/event/BGM24123", nil)
		r.ServeHTTP(w, req)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		t.Logf("Response body: %s", body)

		// Verify title replacement
		if !strings.Contains(body, "<title>Event: Test Event Title</title>") {
			t.Errorf("missing or incorrect title")
		}

		// Verify Open Graph tags
		if !strings.Contains(body, `meta property="og:title" content="Test Event Title"`) {
			t.Errorf("missing og:title")
		}
		if !strings.Contains(body, `meta property="og:description" content="Short Desc"`) {
			t.Errorf("missing og:description")
		}

		// Verify Twitter data (Game system)
		if !strings.Contains(body, `meta property="twitter:label1" content="Game"`) {
			t.Errorf("missing twitter:label1")
		}
		if !strings.Contains(body, `meta property="twitter:data1" content="Catan"`) {
			t.Errorf("missing twitter:data1 content")
		}
	})

	t.Run("Non-event route serves original index", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("context", &Context{})
			c.Next()
		})
		r.NoRoute(ServeV2(db, cache))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/search", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		if w.Body.String() != indexContent {
			t.Errorf("expected original index content, got %s", w.Body.String())
		}
	})

	t.Run("Caching works", func(t *testing.T) {
		// Modify the file on disk
		err := os.WriteFile("static/v2/index.html", []byte("Modified Content"), 0644)
		if err != nil {
			t.Fatalf("failed to write index.html: %s", err)
		}

		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("context", &Context{})
			c.Next()
		})
		r.NoRoute(ServeV2(db, cache))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/some-other-route", nil)
		r.ServeHTTP(w, req)

		// Should still return original content from cache
		if w.Body.String() != indexContent {
			t.Errorf("cache failed, got modified content")
		}
	})

	t.Run("User info injection", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("context", &Context{
				User: &postgres.User{
					Email:       "test@example.com",
					DisplayName: "Test User",
				},
			})
			c.Next()
		})
		r.NoRoute(ServeV2(db, cache))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/search", nil)
		r.ServeHTTP(w, req)

		body := w.Body.String()
		if !strings.Contains(body, `window.serverSideUser = {"displayName":"Test User","email":"test@example.com","genconEmail":"","genconId":"","genconName":""};`) {
			t.Errorf("missing serverSideUser injection. Body: %s", body)
		}
	})
}
