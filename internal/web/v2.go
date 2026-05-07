package web

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
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

		// 1. Check if it's a physical file in static/v2
		// The path will start with /v2/
		filePath := strings.TrimPrefix(path, "/v2")
		filePath = strings.TrimPrefix(filePath, "/")
		
		if filePath != "" {
			fullPath := "static/v2/" + filePath
			if _, err := os.Stat(fullPath); err == nil {
				c.File(fullPath)
				return
			}
		}

		// 2. Check if it's an event route for meta tag injection: /v2/event/:eid
		if strings.HasPrefix(path, "/v2/event/") {
			eid := strings.TrimPrefix(path, "/v2/event/")
			eid = strings.TrimSuffix(eid, "/")

			if len(eid) > 0 {
				serveV2WithMeta(c, db, cache, eid)
				return
			}
		}

		// 3. Fallback: serve the static index.html (SPA entry point)
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

	htmlContent := injectUser(string(content), appContext.User)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
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

	htmlContent := string(content)

	// 3. Render meta tags using the shared Go template
	var metaBuf bytes.Buffer
	t, err := template.New("").Funcs(GetTemplateFunctions(cache)).ParseGlob("templates/*")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	
	err = t.ExecuteTemplate(&metaBuf, "meta", gin.H{
		"event": e,
		"isV2":  true,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Also handle the title tag replacement
	titleTag := fmt.Sprintf("<title>Event: %s</title>\n", html.EscapeString(e.Title))
	metaTags := titleTag + metaBuf.String()

	// 4. Inject tags into <head>
	// We replace the existing title tag and prepend other meta tags before </head>
	if titleRegex.MatchString(htmlContent) {
		htmlContent = titleRegex.ReplaceAllString(htmlContent, metaTags)
	} else {
		htmlContent = strings.Replace(htmlContent, "</head>", metaTags+"</head>", 1)
	}

	// 5. Inject user info
	htmlContent = injectUser(htmlContent, appContext.User)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
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
