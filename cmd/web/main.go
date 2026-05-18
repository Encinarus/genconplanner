package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/api"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/bgg"
	"github.com/Encinarus/genconplanner/internal/logging"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/Encinarus/genconplanner/internal/web"
	"github.com/gin-gonic/gin"
	"github.com/heroku/x/hmetrics"

	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/heroku/x/hmetrics/onload"
	_ "github.com/lib/pq"
	"google.golang.org/api/option"
)

var port = flag.Int("port", 8080, "port to listen on")
var sourceFile = flag.String("eventFile", "https://www.gencon.com/downloads/events.xlsx", "file path or url to load from")

func main() {
	flag.Parse()

	logging.PrintEnv()

	// Don't care about canceling or errors
	go hmetrics.Report(context.Background(), hmetrics.DefaultEndpoint, nil)

	db, err := postgres.OpenDb()
	if err != nil {
		logging.LogWithError(err, "Error opening postgres")
		log.Fatal(err)
	}
	defer db.Close()

	cache := background.NewGameCache(db)
	cache.PeriodicallyUpdate()
	SetupBackground(db)

	SetupWeb(db, cache) // Must be last, won't return until server shutdown
}

func SetupBackground(db *sql.DB) {
	apiKey := os.Getenv("BGG_API_KEY")

	// We run this in a background thread on web because running as a separate
	// app would be expensive. Unlike updating from gencon, these take a long time to
	// process, so the app would be running continually, costing a bit more money than
	// we want.
	// Update from BGG once per week
	bggTicker := time.NewTicker(time.Hour * 24 * 7)

	go func() {
		for {
			// Delay until the next tick
			background.UpdateGamesFromBGG(context.Background(), db, bgg.NewBggApi(apiKey), background.RealClock{})
			select {
			case <-bggTicker.C:
			}
		}
	}()

	genconTicker := time.NewTicker(30 * time.Minute)
	go func() {
		for {
			background.UpdateEventsFromGencon(db, *sourceFile)
			select {
			case <-genconTicker.C:
			}
		}
	}()
}

func SetupWeb(db *sql.DB, cache *background.GameCache) {

	opt := option.WithCredentialsJSON([]byte(os.Getenv("FIREBASE_CONFIG")))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	r := gin.Default()
	r.Use(logging.ErrorStackTrace())
	r.Use(web.BootstrapContext(app, db, cache))

	r.SetFuncMap(web.GetTemplateFunctions(cache))
	r.LoadHTMLGlob("templates/*")

	r.Static("/static/stylesheets", "static/stylesheets")
	r.Static("/static/img", "static/img")
	r.StaticFile("/robots.txt", "static/robots.txt")

	r.GET("/v2/*v2path", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, c.Param("v2path"))
	})
	r.NoRoute(web.ServeV2(db, cache))

	legacy := r.Group("/legacy")
	legacy.Use(web.LegacyCSRFMiddleware())
	legacy.GET("/event/:eid", web.ViewEvent(db))
	legacy.GET("/search", web.Search(db))
	legacy.GET("/cat/:year/:cat", web.ViewCategory(db))
	index := func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("/legacy/cat/%d", time.Now().Year()))
	}
	legacy.GET("/", index)
	legacy.GET("/index", index)
	legacy.GET("/cat/:year", web.CategoryList(db))
	legacy.GET("/starred/:year", web.StarredPage(db))
	legacy.POST("/starEvent/", web.StarEvent(db))
	legacy.GET("/starEvent/", web.GetStarredEvents(db))
	legacy.GET("/listStarredGroups/:year", web.GetStarredEventGroups(db))
	legacy.GET("/about", web.About(db))
	legacy.GET("/user", web.User(db))
	legacy.GET("/admin/orgs/", web.ViewOrgs(db))
	legacy.POST("/admin/orgs/", web.MergeOrgs(db))

	legacy.POST("/party/new", web.NewParty(db))
	legacy.GET("/party/:party_id", web.Party(db))

	repo := &api.PostgresRepository{DB: db}
	api.BuildAPIRoutes(r.Group("/api"), repo, cache, app)

	p := *port
	if os.Getenv("PORT") != "" {
		fmt.Sscanf(os.Getenv("PORT"), "%d", &p)
	}
	r.Run(fmt.Sprintf(":%d", p))
}
