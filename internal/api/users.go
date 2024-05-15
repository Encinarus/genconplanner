package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

type User struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type UserEvents struct {
	Email           string   `json:"email"`
	Year            int      `json:"year"`
	StarredClusters []string `json:"starredClusters"`
	StarredEvents   []string `json:"starredEvents"`
	TicketedEvents  []string `json:"ticketedEvents"`
}

func getFirebaseUser(c *gin.Context, app *firebase.App) (email string, err error) {
	idToken, err := c.Cookie("signinToken")

	if err != nil {
		log.Printf("error getting signin token %v", err)
		return "", err
	}

	ctx := context.Background()
	client, err := app.Auth(ctx)
	if err != nil {
		log.Printf("error getting Auth client: %v\n", err)
		return "", err
	}

	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		log.Printf("error verifying ID token: %v\n", err)
		return "", err
	}

	if token != nil {
		return token.Claims["email"].(string), nil
	}

	return "", nil
}

func requireLogin(c *gin.Context, app *firebase.App) string {
	email, err := getFirebaseUser(c, app)
	if err != nil {
		log.Printf("error getting signin token %v", err)
		c.AbortWithError(http.StatusUnauthorized, err)
		return ""
	} else if email == "" {
		log.Printf("No token, but also no error, unclear what to do")
		c.AbortWithError(http.StatusUnauthorized, err)
		return ""
	}

	return email
}

func getUser(c *gin.Context, db *sql.DB, app *firebase.App) {
	email := requireLogin(c, app)
	if email == "" {
		// requireLogin already aborted the request.
		return
	}

	dbUser, err := postgres.LoadOrCreateUser(db, email)
	if err != nil {
		log.Printf("error loading/creating user: %v\n", err)
		c.AbortWithError(http.StatusServiceUnavailable, err)
		return
	}

	var user User
	user.DisplayName = dbUser.DisplayName
	user.Email = dbUser.Email
	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.Writer).Encode(user)

}

func loadUserEvents(c *gin.Context, db *sql.DB, app *firebase.App) {
	email := requireLogin(c, app)
	if email == "" {
		// requireLogin already aborted the request.
		return
	}

	// TODO: factor in user email and year from params
	var userEvents UserEvents
	starredIds, err := postgres.GetStarredIds(db, email)
	if err != nil {
		log.Printf("error getting user starred list: %v\n", err)
	} else {
		for _, starred := range starredIds.StarredEvents {
			if starred.Level == "group" {
				userEvents.StarredClusters = append(userEvents.StarredClusters, starred.EventId)
			} else if starred.Level == "event" {
				userEvents.StarredEvents = append(userEvents.StarredEvents, starred.EventId)
			}
		}
	}

	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.Writer).Encode(userEvents)
}

func userRoutes(api_group *gin.RouterGroup, db *sql.DB, app *firebase.App) {
	api_group.GET("/user/", func(c *gin.Context) {
		getUser(c, db, app)
	})
	api_group.GET("/user/events/:email/:year", func(c *gin.Context) {
		loadUserEvents(c, db, app)
	})
}
