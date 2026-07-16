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
	go func() {
		_ = hmetrics.Report(context.Background(), hmetrics.DefaultEndpoint, nil)
	}()

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
			<-bggTicker.C
		}
	}()

	genconTicker := time.NewTicker(30 * time.Minute)
	go func() {
		for {
			background.UpdateEventsFromGencon(db, *sourceFile)
			<-genconTicker.C
		}
	}()
}

func SetupWeb(db *sql.DB, cache *background.GameCache) {

	//nolint:staticcheck // option.WithCredentialsJSON is deprecated but compatible with this configuration
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


	repo := &api.PostgresRepository{DB: db}
	api.BuildAPIRoutes(r.Group("/api"), repo, cache, app)

	p := *port
	if os.Getenv("PORT") != "" {
		if _, err := fmt.Sscanf(os.Getenv("PORT"), "%d", &p); err != nil {
			log.Printf("Error parsing PORT: %v", err)
		}
	}
	if err := r.Run(fmt.Sprintf(":%d", p)); err != nil {
		log.Fatalf("Error running server: %v", err)
	}
}
