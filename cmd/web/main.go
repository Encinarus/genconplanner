package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
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

	// Don't care about canceling or errors
	go hmetrics.Report(context.Background(), hmetrics.DefaultEndpoint, nil)

	db, err := postgres.OpenDb()
	if err != nil {
		log.Println("Error opening postgres")
		log.Fatal(err)
	}
	defer db.Close()

	cache := background.NewGameCache(db)
	cache.PeriodicallyUpdate()
	SetupBackground(db)

	SetupWeb(db, cache) // Must be last, won't return until server shutdown
}

func SetupBackground(db *sql.DB) {
	// We run this in a background thread on web because running as a separate
	// app would be expensive. Unlike updating from gencon, these take a long time to
	// process, so the app would be running continually, costing a bit more money than
	// we want.
	// Update from BGG once per week
	bggTicker := time.NewTicker(time.Hour * 24 * 7)

	go func() {
		for {
			// Delay until the next tick
			background.UpdateGamesFromBGG(db)
			select {
			case <-bggTicker.C:
			}
		}
	}()

	// TODO: decide if this should run here too.
	if 1 == 2 {
		genconTicker := time.NewTicker(time.Hour)
		go func() {
			for {
				background.UpdateEventsFromGencon(db, *sourceFile)
				select {
				case <-genconTicker.C:
				}
			}
		}()
	}
}

func SetupWeb(db *sql.DB, cache *background.GameCache) {

	opt := option.WithCredentialsJSON([]byte(os.Getenv("FIREBASE_CONFIG")))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	r := gin.Default()
	r.Use(web.BootstrapContext(app, db, cache))

	r.SetFuncMap(web.GetTemplateFunctions(cache))
	r.LoadHTMLGlob("templates/*")

	r.Static("/static/stylesheets", "static/stylesheets")
	r.Static("/static/img", "static/img")
	r.StaticFile("/robots.txt", "static/robots.txt")

	r.GET("/event/:eid", web.ViewEvent(db))
	r.GET("/search", web.Search(db))
	r.GET("/cat/:year/:cat", web.ViewCategory(db))
	index := func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("/cat/%d", time.Now().Year()))
	}
	r.GET("/", index)
	r.GET("/index", index)
	r.GET("/cat/:year", web.CategoryList(db))
	r.GET("/starred/:year", web.StarredPage(db))
	r.POST("/starEvent/", web.StarEvent(db))
	r.GET("/starEvent/", web.GetStarredEvents(db))
	r.GET("/listStarredGroups/:year", web.GetStarredEventGroups(db))
	r.GET("/about", web.About(db))
	r.GET("/user", web.User(db))
	r.GET("/admin/orgs/", web.ViewOrgs(db))
	r.POST("/admin/orgs/", web.MergeOrgs(db))

	r.POST("/party/new", web.NewParty(db))
	r.GET("/party/:party_id", web.Party(db))
	r.Run(fmt.Sprintf(":%d", *port))
}
