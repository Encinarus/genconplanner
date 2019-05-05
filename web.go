package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"firebase.google.com/go"
	"flag"
	"fmt"
	"github.com/Encinarus/genconplanner/events"
	"github.com/Encinarus/genconplanner/postgres"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var port = flag.Int("port", 8080, "port to listen on")

type Context struct {
	Year     int
	Username string
}

type LookupResult struct {
	MainEvent    *events.GenconEvent
	Wednesday    []*events.SlimEvent
	Thursday     []*events.SlimEvent
	Friday       []*events.SlimEvent
	Saturday     []*events.SlimEvent
	Sunday       []*events.SlimEvent
	TotalTickets int
}

func LookupEvent(db *sql.DB, eventId string) *LookupResult {
	foundEvents, err := postgres.LoadSimilarEvents(db, eventId)
	if err != nil {
		log.Fatalf("Unable to load events, err %v", err)
	}
	log.Printf("Found %v events similar to %s", len(foundEvents), eventId)

	var result LookupResult
	for _, event := range foundEvents {
		if event.EventId == eventId {
			result.MainEvent = event
		}

		switch event.StartTime.Weekday() {
		case time.Wednesday:
			result.Wednesday = append(result.Wednesday, event.SlimEvent())
			break
		case time.Thursday:
			result.Thursday = append(result.Thursday, event.SlimEvent())
			break
		case time.Friday:
			result.Friday = append(result.Friday, event.SlimEvent())
			break
		case time.Saturday:
			result.Saturday = append(result.Saturday, event.SlimEvent())
			break
		case time.Sunday:
			result.Sunday = append(result.Sunday, event.SlimEvent())
			break
		}

		result.TotalTickets += event.TicketsAvailable
	}

	return &result
}

func CategoryList(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		defaultYear := time.Now().Year()

		var err error
		context := c.MustGet("context").(*Context)

		if len(strings.TrimSpace(c.Param("year"))) > 0 {
			context.Year, err = strconv.Atoi(c.Param("year"))
			if err != nil {
				log.Printf("Error parsing year")
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		} else {
			context.Year = defaultYear
		}

		summary, err := postgres.LoadCategorySummary(db, context.Year)

		if err != nil {
			log.Printf("Error loading categories, %v", err)
			c.AbortWithError(500, err)
			return
		}

		batchSize := 2
		tail := len(summary) % batchSize
		numBuckets := len(summary) / batchSize
		if tail > 0 {
			numBuckets++
		}
		categories := make([][]*postgres.CategorySummary, numBuckets)
		for i := range categories {
			base := batchSize * i
			end := base + batchSize
			if i == len(categories)-1 {
				end = base + tail
			}
			categories[i] = summary[base:end]
		}
		log.Printf("Loaded %d categories in %d rows", len(summary), len(categories))
		c.HTML(http.StatusOK, "categories.html", gin.H{
			"title":      "Main website",
			"categories": categories,
			"context":    context,
		})
	}
}

func parseQuery(searchQuery string, yearParam string) *postgres.ParsedQuery {
	query := postgres.ParsedQuery{}

	query.Year = time.Now().Year()

	yearParam = strings.TrimSpace(yearParam)
	if len(yearParam) > 0 {
		// Silently ignore invalid year params
		parsedYear, err := strconv.Atoi(yearParam)
		if err == nil {
			query.Year = parsedYear
		}
	}

	// Preprocess, removing symbols which are used in tsquery
	searchQuery = strings.Replace(searchQuery, "!", "", -1)
	searchQuery = strings.Replace(searchQuery, "&", "", -1)
	searchQuery = strings.Replace(searchQuery, "(", "", -1)
	searchQuery = strings.Replace(searchQuery, ")", "", -1)
	searchQuery = strings.Replace(searchQuery, "|", "", -1)

	queryReader := csv.NewReader(bytes.NewBufferString(searchQuery))
	queryReader.Comma = ' '

	splitQuery, _ := queryReader.Read()

	// TODO(alek): consider adding a db field "searchable_text" rather than relying
	// the trigger across many fields. Then exact matches do like vs that, while fuzzy
	// matches go against the ts_vector column
	for _, term := range splitQuery {
		invertTerm := false
		if strings.HasPrefix(term, "-") {
			log.Println("Negated term:", term)
			term = strings.TrimLeft(term, "-")
			invertTerm = true
		}
		if strings.ContainsAny(term, ":<>=-~") {
			// TODO(alek) Handle key:value searches
			// : and = work as equals
			// < > compare for dates or num tickets
			// ~ is for checking if the string is in a field
			continue
		}

		// Now remove remaining symbols we want to allow in field-specific
		// searches, but not in the general text search
		term = strings.Replace(term, "<", "", -1)
		term = strings.Replace(term, ">", "", -1)
		term = strings.Replace(term, "=", "", -1)
		term = strings.Replace(term, "-", "", -1)
		term = strings.Replace(term, "~", "", -1)
		term = strings.TrimSpace(term)
		if len(term) == 0 {
			continue
		}
		if invertTerm {
			term = "!" + term
		}
		query.TextQueries = append(query.TextQueries, term)
	}
	return &query
}

func Search(db *sql.DB) func(c *gin.Context) {
	keyFunc := func(g *postgres.EventGroup) string {
		if len(strings.TrimSpace(g.ShortCategory)) == 0 {
			return "Unknown"
		}
		longCat, found := map[string]string{
			"ANI":  "Anime Activities",
			"BGM":  "Board Games",
			"CGM":  "Non-Collectable/Tradable Card Games",
			"EGM":  "Electronic Games",
			"ENT":  "Entertainment Events",
			"FLM":  "Film Fest",
			"HMN":  "Historical Miniatures",
			"KID":  "Kids Activities",
			"LRP":  "Larps",
			"MHE":  "Miniature Hobby Events",
			"NMN":  "Non-Historical Miniatures",
			"RPG":  "Role Playing Games",
			"RPGA": "Role Playing Game Association",
			"SEM":  "Seminiars",
			"SPA":  "Spousal Activities",
			"TCG":  "Tradeable Card Game",
			"TDA":  "True Dungeon",
			"TRD":  "Trade Day Events",
			"WKS":  "Workshop",
			"ZED":  "Isle of Misfit Events",
		}[g.ShortCategory]

		if found {
			return longCat
		} else {
			return g.ShortCategory
		}
	}

	return func(c *gin.Context) {
		query := c.Query("q")
		year := c.Query("y")
		log.Println("Raw Query: ", query)

		parsedQuery := parseQuery(query, year)

		eventGroups, err := postgres.FindEvents(db, parsedQuery)
		totalEvents := 0
		for _, group := range eventGroups {
			totalEvents += group.Count
		}
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		} else {
			headings, partitions := partitionGroups(eventGroups, keyFunc)
			c.HTML(http.StatusOK, "results.html", gin.H{
				"year":        year,
				"headings":    headings,
				"partitions":  partitions,
				"totalEvents": totalEvents,
				"groups":      len(eventGroups),
				"breakdown":   "Category",
				"pageHeader":  "Search",
				"subHeader":   query,
			})
		}
	}
}

func partitionGroups(
	groups []*postgres.EventGroup,
	keyFunction func(*postgres.EventGroup) string,
) ([]string, map[string][]*postgres.EventGroup) {

	partitions := make(map[string][]*postgres.EventGroup)
	keys := make([]string, 0)

	for _, group := range groups {
		key := keyFunction(group)
		partition, ok := partitions[key]
		if !ok {
			partition = make([]*postgres.EventGroup, 0)
			keys = append(keys, key)
		}
		partitions[key] = append(partition, group)
	}
	sort.Strings(keys)
	return keys, partitions
}

func ViewCategory(db *sql.DB) func(c *gin.Context) {
	keyFunc := func(g *postgres.EventGroup) string {
		if len(strings.TrimSpace(g.GameSystem)) == 0 {
			return "Unspecified"
		}
		return g.GameSystem
	}
	return func(c *gin.Context) {
		appContext := c.MustGet("context").(*Context)
		var err error

		appContext.Year, err = strconv.Atoi(c.Param("year"))
		if err != nil {
			log.Printf("Error parsing year")
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		cat := c.Param("cat")
		if len(strings.TrimSpace(cat)) == 0 {
			log.Printf("No category specified")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		eventGroups, err := postgres.LoadEventGroups(db, c.Param("cat"), appContext.Year)
		if err != nil {
			log.Printf("Error loading event groups")
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		totalEvents := 0
		for _, group := range eventGroups {
			totalEvents += group.Count
		}

		headings, partitions := partitionGroups(eventGroups, keyFunc)
		c.HTML(http.StatusOK, "results.html", gin.H{
			"context":     appContext,
			"headings":    headings,
			"partitions":  partitions,
			"totalEvents": totalEvents,
			"groups":      len(eventGroups),
			"breakdown":   "Systems",
			"pageHeader":  "Category",
			"subHeader":   cat,
		})
	}
}

func bootstrapContext(app *firebase.App) gin.HandlerFunc {
	return func(c *gin.Context) {
		var appContext Context
		// Process login: signinToken

		// Create user if needed based on cookie

		idToken, err := c.Cookie("signinToken")
		if err == nil {
			log.Println("Signin token:", idToken)

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
			email := token.Claims["email"]

			appContext.Username = strings.Split(email.(string), "@")[0]
			log.Printf("Username %v\n", appContext)
		}

		c.Set("context", &appContext)

		c.Next()
	}
}

func main() {
	flag.Parse()

	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}
	log.Print(app)

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

	db, err := postgres.OpenDb()
	if err != nil {
		log.Println("Error opening postgres")
		log.Fatal(err)
	}
	defer db.Close()

	r := gin.Default()
	r.Use(bootstrapContext(app))
	r.SetFuncMap(template.FuncMap{
		"toId": textToId,
		"dict": dict,
	})
	r.LoadHTMLGlob("templates/*")
	r.Static("/static/stylesheets", "static/stylesheets")
	categoryList := CategoryList(db)

	r.GET("/event/:eid", func(c *gin.Context) {
		eventId := c.Param("eid")
		result := LookupEvent(db, eventId)
		c.HTML(http.StatusOK, "event.html", gin.H{
			"year":   result.MainEvent.Year,
			"result": result,
		})
	})
	r.GET("/search", Search(db))
	r.GET("/cat/:year/:cat", ViewCategory(db))
	index := func(c *gin.Context) {
		year := time.Now().Year()
		c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/cat/%d", year))
	}
	r.GET("/", index)
	r.GET("/index", index)
	r.GET("/cat/:year", categoryList)
	r.GET("/about", func(c *gin.Context) {
		year, err := strconv.Atoi(c.Param("year"))
		if err != nil {
			year = time.Now().Year()
		}
		appContext := c.MustGet("context").(*Context)
		appContext.Year = year
		c.HTML(http.StatusOK, "about.html", gin.H{
			"context": appContext,
		})
	})
	r.Run(fmt.Sprintf(":%d", *port))
}
