package main

import (
	"context"
	"database/sql"
	"firebase.google.com/go"
	"flag"
	"fmt"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/Encinarus/genconplanner/internal/web"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/api/option"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var port = flag.Int("port", 8080, "port to listen on")
var sourceFile = flag.String("eventFile", "https://www.gencon.com/downloads/events.xlsx", "file path or url to load from")

func main() {
	flag.Parse()

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
			background.UpdateGamesFromBGG(db)
			// Delay until the next tick
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
	textStrippingRegex, _ := regexp.Compile("[^a-zA-Z0-9]+")
	textToId := func(text string) string {
		return textStrippingRegex.ReplaceAllString(strings.ToLower(text), "")
	}
	dict := func(v ...interface{}) map[string]interface{} {
		dict := map[string]interface{}{}
		lenv := len(v)
		for i := 0; i < lenv; i += 2 {
			key := fmt.Sprintf("%s", v[i])
			if i+1 >= lenv {
				dict[key] = ""
				continue
			}
			dict[key] = v[i+1]
		}
		return dict
	}

	bggPage := func(gameName string) string {
		bggGames := cache.FindGame(gameName)
		if len(bggGames) == 0 {
			return ""
		}

		// We'll just use the first one. Hopefully conflicts don't actually come up in practice
		return fmt.Sprintf("https://boardgamegeek.com/boardgame/%d", bggGames[0].BggId)
	}

	bggRating := func(gameName string) string {
		bggGames := cache.FindGame(gameName)
		if len(bggGames) == 0 || bggGames[0].AvgRatings < 0.1 {
			return ""
		}

		// We'll just use the first one. Hopefully conflicts don't actually come up in practice
		return fmt.Sprintf("%2.1f", bggGames[0].AvgRatings)
	}

	opt := option.WithCredentialsJSON([]byte(os.Getenv("FIREBASE_CONFIG")))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	r := gin.Default()
	r.Use(bootstrapContext(app, db))

	r.SetFuncMap(template.FuncMap{
		"toId":      textToId,
		"dict":      dict,
		"bggPage":   bggPage,
		"bggRating": bggRating,
	})
	r.LoadHTMLGlob("templates/*")

	r.Static("/static/stylesheets", "static/stylesheets")
	r.StaticFile("/robots.txt", "static/robots.txt")

	r.GET("/event/:eid", web.ViewEvent(db))
	r.GET("/search", web.Search(db))
	r.GET("/cat/:year/:cat", web.ViewCategory(db))
	index := func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently,
			fmt.Sprintf("/cat/%d", time.Now().Year()))
	}
	r.GET("/", index)
	r.GET("/index", index)
	r.GET("/cat/:year", web.CategoryList(db))
	r.GET("/starred/:year", web.StarredPage(db))
	r.POST("/starEvent/", web.StarEvent(db))
	r.GET("/starEvent/", web.GetStarredEvents(db))
	r.GET("/listStarredGroups/:year", web.GetStarredEventGroups(db))

	r.GET("/about", func(c *gin.Context) {
		year, err := strconv.Atoi(c.Param("year"))
		if err != nil {
			year = time.Now().Year()
		}
		appContext := c.MustGet("context").(*web.Context)
		appContext.Year = year
		c.HTML(http.StatusOK, "about.html", gin.H{
			"context": appContext,
		})
	})
	r.Run(fmt.Sprintf(":%d", *port))
}

func bootstrapContext(app *firebase.App, db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var appContext web.Context
		appContext.Starred = &postgres.UserStarredEvents{}

		if c.Request.UserAgent() != "" {
			log.Printf("UserAgent: %v\n", c.Request.UserAgent())
		}
		// Create user if needed based on cookie
		idToken, err := c.Cookie("signinToken")
		if err == nil {
			ctx := context.Background()
			client, err := app.Auth(ctx)
			if err != nil {
				log.Printf("error getting Auth client: %v\n", err)
				return
			}
			token, err := client.VerifyIDToken(ctx, idToken)
			if err != nil {
				log.Printf("error verifying ID token: %v\n", err)
			}
			if token != nil {
				email := token.Claims["email"].(string)

				appContext.Email = email
				user, err := postgres.LoadOrCreateUser(db, email)
				if err != nil {
					log.Printf("Error Loading/creating user: %v\n", err)
				} else {
					if user.DisplayName == "" {
						user.DisplayName = strings.Split(email, "@")[0]
					}
					appContext.DisplayName = user.DisplayName
					if err != nil {
						log.Printf("error loading starred events for user %v\n", err)
					}
				}
			}
		}

		c.Set("context", &appContext)
		c.Next()
	}
}
