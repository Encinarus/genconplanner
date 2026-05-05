package api

import (
	"log"
	"net/http"

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


func getUser(c *gin.Context, repo EventRepository) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	dbUser, err := repo.LoadOrCreateUser(email)
	if err != nil {
		log.Printf("error loading/creating user: %v\n", err)
		c.AbortWithError(http.StatusServiceUnavailable, err)
		return
	}

	var user User
	user.DisplayName = dbUser.DisplayName
	user.Email = dbUser.Email
	c.JSON(http.StatusOK, user)
}

func loadUserEvents(c *gin.Context, repo EventRepository) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// TODO: factor in user email and year from params
	var userEvents UserEvents
	starredIds, err := repo.GetStarredIds(email)
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

	c.JSON(http.StatusOK, userEvents)
}

func userRoutes(api_group *gin.RouterGroup, repo EventRepository) {
	api_group.GET("/user/", func(c *gin.Context) {
		getUser(c, repo)
	})
	api_group.GET("/user/events/:email/:year", func(c *gin.Context) {
		loadUserEvents(c, repo)
	})
}
