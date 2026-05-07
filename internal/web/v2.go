package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

var (
	v2IndexCache []byte
	v2IndexMu    sync.RWMutex
	titleRegex   = regexp.MustCompile(`(?i)<title>.*?</title>`)
)

// ServeV2 returns a handler that serves the new UI (v2).
// It specifically intercepts event routes to inject dynamic Open Graph meta tags
// for better social media previews (Slack, Twitter, etc.).
func ServeV2(db *sql.DB, cache *background.GameCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Only handle /v2 routes
		if !strings.HasPrefix(path, "/v2") {
			return
		}

		// Check if it's an event route: /v2/event/:eid
		if strings.HasPrefix(path, "/v2/event/") {
			eid := strings.TrimPrefix(path, "/v2/event/")
			eid = strings.TrimSuffix(eid, "/")

			if len(eid) > 0 {
				serveV2WithMeta(c, db, cache, eid)
				return
			}
		}

		// Fallback: serve the static index.html
		serveV2Static(c)
	}
}

func serveV2Static(c *gin.Context) {
	appContext := c.MustGet("context").(*Context)
	content, err := getV2Index()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	html := injectUser(string(content), appContext.User)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func getV2Index() ([]byte, error) {
	v2IndexMu.RLock()
	if v2IndexCache != nil {
		defer v2IndexMu.RUnlock()
		return v2IndexCache, nil
	}
	v2IndexMu.RUnlock()

	v2IndexMu.Lock()
	defer v2IndexMu.Unlock()

	// Double check after acquiring write lock
	if v2IndexCache != nil {
		return v2IndexCache, nil
	}

	content, err := os.ReadFile("static/v2/index.html")
	if err != nil {
		return nil, err
	}
	v2IndexCache = content
	return v2IndexCache, nil
}

func serveV2WithMeta(c *gin.Context, db *sql.DB, cache *background.GameCache, eid string) {
	// 1. Fetch event details
	appContext := c.MustGet("context").(*Context)
	result, err := lookupEvent(db, eid, appContext.Email)
	if err != nil || result == nil || result.MainEvent == nil {
		// If event not found, just serve the static SPA
		serveV2Static(c)
		return
	}
	e := result.MainEvent

	// 2. Get base index.html content
	content, err := getV2Index()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	html := string(content)

	// 3. Construct meta tags
	var meta strings.Builder
	title := fmt.Sprintf("Event: %s", e.Title)

	meta.WriteString(fmt.Sprintf("<title>%s</title>\n", title))
	meta.WriteString(fmt.Sprintf("<meta property=\"og:title\" content=\"%s\" />\n", e.Title))
	meta.WriteString(fmt.Sprintf("<meta property=\"og:description\" content=\"%s\" />\n", e.ShortDescription))
	meta.WriteString("<meta property=\"og:type\" content=\"article\" />\n")

	if e.GameSystem != "" {
		gameLabel := e.GameSystem
		year := bggYear(e.GameSystem, cache)
		if year != "" {
			gameLabel = fmt.Sprintf("%s (%s)", gameLabel, year)
		}
		meta.WriteString(fmt.Sprintf("<meta property=\"twitter:label1\" content=\"Game\" />\n"))
		meta.WriteString(fmt.Sprintf("<meta property=\"twitter:data1\" content=\"%s\" />\n", gameLabel))

		rating := bggRating(e.GameSystem, cache)
		numRatings := bggNumRatings(e.GameSystem, cache)
		if rating != "" && numRatings != "" {
			meta.WriteString(fmt.Sprintf("<meta property=\"twitter:label2\" content=\"BGG Rating\" />\n"))
			meta.WriteString(fmt.Sprintf("<meta property=\"twitter:data2\" content=\"%s with %s ratings\" />\n", rating, numRatings))
		}
	}

	// 4. Inject tags into <head>
	// We replace the existing title tag and prepend other meta tags before </head>
	if titleRegex.MatchString(html) {
		html = titleRegex.ReplaceAllString(html, meta.String())
	} else {
		html = strings.Replace(html, "</head>", meta.String()+"</head>", 1)
	}

	// 5. Inject user info
	html = injectUser(html, appContext.User)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func injectUser(html string, user *postgres.User) string {
	if user == nil {
		return html
	}

	userJson, err := json.Marshal(map[string]string{
		"email":       user.Email,
		"displayName": user.DisplayName,
	})
	if err != nil {
		return html
	}

	script := fmt.Sprintf("<script>window.serverSideUser = %s;</script>\n", string(userJson))
	return strings.Replace(html, "</head>", script+"</head>", 1)
}
