package web

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

type FirebaseConfig struct {
	ApiKey            string
	AuthDomain        string
	DatabaseURL       string
	MessagingSenderId string
	ProjectId         string
	StorageBucket     string
}

type Context struct {
	Year        int
	DisplayName string
	Email       string
	Starred     *postgres.UserStarredEvents
	User        *postgres.User
	Firebase    FirebaseConfig
	BggCache    *background.GameCache
}

type EventKeyFunc func(*postgres.EventGroup, *Context) (string, string)

type QueryParams struct {
	Year            int
	Days            map[string]bool
	StartBeforeHour int
	StartAfterHour  int
	EndBeforeHour   int
	EndAfterHour    int
	Grouping        EventKeyFunc
	SortAsc         bool
	Query           string
	OrgId           int
	Category        string
}

func caseInsensitiveSort(data []string) {
	slices.SortFunc(data, func(a, b string) int {
		return cmp.Compare(strings.ToLower(a), strings.ToLower(b))
	})
}

func parseQuery(params QueryParams) *postgres.ParsedQuery {
	query := postgres.ParsedQuery{
		Year:            params.Year,
		DaysOfWeek:      params.Days,
		RawQuery:        params.Query,
		OrgId:           params.OrgId,
		StartBeforeHour: params.StartBeforeHour,
		StartAfterHour:  params.StartAfterHour,
		EndBeforeHour:   params.EndBeforeHour,
		EndAfterHour:    params.EndAfterHour,
	}

	maxYear := time.Now().Year()
	// This version of genconplanner didn't exist before 2019
	if query.Year > maxYear || query.Year < 2019 {
		query.Year = maxYear
	}

	log.Printf("Search query: %v", query)

	// Preprocess, removing symbols which are used in tsquery
	params.Query = strings.Replace(params.Query, "!", "", -1)
	params.Query = strings.Replace(params.Query, "&", "", -1)
	params.Query = strings.Replace(params.Query, "(", "", -1)
	params.Query = strings.Replace(params.Query, ")", "", -1)
	params.Query = strings.Replace(params.Query, "|", "", -1)

	queryReader := csv.NewReader(bytes.NewBufferString(params.Query))
	queryReader.Comma = ' '

	splitQuery, _ := queryReader.Read()

	for _, term := range splitQuery {
		invertTerm := false
		if strings.HasPrefix(term, "-") {
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

func processQueryParams(c *gin.Context) QueryParams {
	var params QueryParams
	var err error

	params.Query = c.Query("q")

	// First query, then context, then now
	params.Year, err = strconv.Atoi(c.Query("year"))
	if err != nil {
		params.Year, err = strconv.Atoi(c.Param("year"))
		if err != nil {
			params.Year = time.Now().Year()
		}
	}

	params.Category = strings.TrimSpace(c.Param("cat"))

	groupMethod := c.Query("grouping")
	switch groupMethod {
	case "org":
		params.Grouping = KeyByCategoryOrg
	case "sys":
		params.Grouping = KeyByCategorySystem
	case "bgg":
		params.Grouping = KeyByBggYear
	default:
		params.Grouping = KeyByCategorySystem
	}

	_, found := c.GetQuery("sortDesc")
	params.SortAsc = !found

	params.Days = make(map[string]bool)
	for _, day := range []string{"wed", "thu", "fri", "sat", "sun"} {
		param, found := c.GetQuery(day)

		if found && len(param) > 0 {
			if b, err := strconv.ParseBool(param); err == nil {
				params.Days[day] = b
			}
		}
	}

	orgId, err := strconv.Atoi(c.Query("org_id"))
	if err == nil {
		params.OrgId = orgId
	}

	params.StartBeforeHour = parseHour(c, "start_before", -1)
	params.StartAfterHour = parseHour(c, "start_after", -1)
	params.EndBeforeHour = parseHour(c, "end_before", -1)
	params.EndAfterHour = parseHour(c, "end_after", -1)
	if params.StartBeforeHour == params.StartAfterHour {
		params.StartBeforeHour = -1
		params.StartAfterHour = -1
	}
	if params.EndAfterHour == params.EndBeforeHour {
		params.EndAfterHour = -1
		params.EndBeforeHour = -1
	}

	return params
}

func KeyByCategorySystem(g *postgres.EventGroup, context *Context) (majorGroup, minorGroup string) {
	majorGroup = events.LongCategory(g.ShortCategory)
	minorGroup = "Unspecified"

	if len(strings.TrimSpace(g.GameSystem)) != 0 {
		minorGroup = strings.TrimSpace(g.GameSystem)
	}

	return majorGroup, minorGroup
}

func KeyByCategoryOrg(g *postgres.EventGroup, context *Context) (majorGroup, minorGroup string) {
	majorGroup = events.LongCategory(g.ShortCategory)
	minorGroup = "Unknown Organizer"

	if len(strings.TrimSpace(g.OrgGroup)) != 0 {
		minorGroup = g.OrgGroup
	}

	return majorGroup, minorGroup
}

func KeyByBggYear(g *postgres.EventGroup, context *Context) (majorGroup, minorGroup string) {
	game := context.BggCache.FindGame(g.GameSystem)
	majorGroup = "Unknown Year"
	if game != nil && game.YearPublished > 0 {
		majorGroup = fmt.Sprintf("Published %d", game.YearPublished)
	}
	minorGroup = g.GameSystem

	return majorGroup, minorGroup
}

func BootstrapContext(app *firebase.App, db *sql.DB, bggCache *background.GameCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		var appContext Context
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
					appContext.User = user
					if user.DisplayName == "" {
						user.DisplayName = strings.Split(email, "@")[0]
					}
					appContext.DisplayName = user.DisplayName
				}
			}
		}

		appContext.Firebase = getFirebaseConfig()
		appContext.BggCache = bggCache
		log.Println("Setting context")

		c.Set("context", &appContext)
		c.Next()
	}
}

func PartitionGroups(
	groups []*postgres.EventGroup,
	context *Context,
	params QueryParams,
) ([]string, map[string][]string, map[string]map[string][]*postgres.EventGroup) {

	majorPartitions := make(map[string]map[string][]*postgres.EventGroup)
	majorKeys := make([]string, 0)
	minorKeys := make(map[string][]string)

	const soldOut = "Sold out"
	hasSoldOut := false

	for _, group := range groups {
		majorKey, minorKey := params.Grouping(group, context)
		if group.TotalTickets == 0 {
			minorKey = majorKey
			majorKey = soldOut
			hasSoldOut = true
		}
		if _, found := majorPartitions[majorKey]; !found {
			majorPartitions[majorKey] = make(map[string][]*postgres.EventGroup)
			majorKeys = append(majorKeys, majorKey)
			minorKeys[majorKey] = make([]string, 0)
		}
		if _, found := majorPartitions[majorKey][minorKey]; !found {
			majorPartitions[majorKey][minorKey] = make([]*postgres.EventGroup, 0)
			// First time encountering this minor key, add to the list
			minorKeys[majorKey] = append(minorKeys[majorKey], minorKey)
		}
		majorPartitions[majorKey][minorKey] = append(majorPartitions[majorKey][minorKey], group)
	}
	caseInsensitiveSort(majorKeys)
	if !params.SortAsc {
		// What I want is to be able to say, sort.String(sort.Reverse(majorKeys))
		// Unfortunately, go is kind of dumb about this. Like really dumb. []string
		// doesn't implement the functions needed for that to work. Wtf.
		numKeys := len(majorKeys)
		for i := 0; i < numKeys/2; i++ {
			base := i
			swap := numKeys - i - 1
			majorKeys[base], majorKeys[swap] = majorKeys[swap], majorKeys[base]
		}
	}

	for k := range minorKeys {
		caseInsensitiveSort(minorKeys[k])
	}
	// Now that we've sorted, move sold out to the end
	if hasSoldOut && len(majorKeys) > 1 {
		index := sort.SearchStrings(majorKeys, soldOut)
		majorKeys = append(majorKeys[:index], majorKeys[index+1:]...)
		majorKeys = append(majorKeys, soldOut)
	}
	return majorKeys, minorKeys, majorPartitions
}

var genconDates = map[int][]string{
	2018: {"2018-08-01", "2018-08-05"},
	2019: {"2019-07-31", "2019-08-04"},
	2020: {"2020-07-29", "2020-08-02"},
	2021: {"2021-09-15", "2021-09-19"},
	2022: {"2022-08-03", "2022-08-07"},
	2023: {"2023-08-02", "2023-08-06"},
	2024: {"2024-07-31", "2024-08-04"},
	2025: {"2025-07-30", "2025-08-03"},
}

func GenconStartDate(year int) string {
	dates, found := genconDates[year]
	if !found {
		return "2019-07-31"
	}

	return dates[0]
}

func GenconEndDate(year int) string {
	dates, found := genconDates[year]
	if !found {
		return "2019-08-04"
	}
	return dates[1]
}

func getEnvWithDefault(key, dflt string) string {
	if val, set := os.LookupEnv(key); set && len(val) > 0 {
		return val
	}
	return dflt
}

func getFirebaseConfig() FirebaseConfig {
	return FirebaseConfig{
		ApiKey:            getEnvWithDefault("FIREBASE_API_KEY", "AIzaSyAGtjwGiHYFnXE1UbzLTPeIz8Ix06WIdBs"),
		AuthDomain:        getEnvWithDefault("FIREBASE_AUTH_DOMAIN", "genconplanner-v2.firebaseapp.com"),
		DatabaseURL:       getEnvWithDefault("FIREBASE_DATABASE_URL", "https://genconplanner-v2.firebaseio.com"),
		ProjectId:         getEnvWithDefault("FIREBASE_PROJECT_ID", "genconplanner-v2"),
		StorageBucket:     getEnvWithDefault("FIREBASE_STORAGE_BUCKET", "genconplanner-v2.appspot.com"),
		MessagingSenderId: getEnvWithDefault("FIREBASE_MESSAGING_SENDER_ID", "630743534199"),
	}
}

func parseHour(c *gin.Context, param string, defaultValue int) int {
	raw, found := c.GetQuery(param)
	if !found {
		return defaultValue
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	} else if parsed < 0 || parsed > 24 {
		return defaultValue
	} else {
		return parsed
	}
}
