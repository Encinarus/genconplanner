package web

import (
	"context"
	"database/sql"
	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"sort"
	"strings"
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
}

func BootstrapContext(app *firebase.App, db *sql.DB) gin.HandlerFunc {
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

		c.Set("context", &appContext)
		c.Next()
	}
}

func PartitionGroups(
	groups []*postgres.EventGroup,
	keyFunction func(*postgres.EventGroup) (string, string),
) ([]string, map[string][]string, map[string]map[string][]*postgres.EventGroup) {

	majorPartitions := make(map[string]map[string][]*postgres.EventGroup)
	majorKeys := make([]string, 0)
	minorKeys := make(map[string][]string)

	const soldOut = "Sold out"
	hasSoldOut := false

	for _, group := range groups {
		majorKey, minorKey := keyFunction(group)
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
	sort.Strings(majorKeys)
	for k := range minorKeys {
		sort.Strings(minorKeys[k])
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
	if val, set := os.LookupEnv(key); set {
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
