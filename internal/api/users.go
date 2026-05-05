package api

import (
	"log"
	"net/http"
	"strconv"

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


func (s *Server) GetUser(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	dbUser, err := s.Repo.LoadOrCreateUser(email)
	if err != nil {
		log.Printf("error loading/creating user: %v\n", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, ErrorResponse{Error: "Service Unavailable"})
		return
	}

	var user User
	user.DisplayName = dbUser.DisplayName
	user.Email = dbUser.Email
	c.JSON(http.StatusOK, user)
}

func (s *Server) LoadUserEvents(c *gin.Context) {
	authedEmail := GetUserEmail(c)
	paramEmail := c.Param("email")
	yearParam := c.Param("year")

	if authedEmail == "" || authedEmail != paramEmail {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	var userEvents UserEvents
	userEvents.Email = authedEmail
	userEvents.Year = year

	starredIds, err := s.Repo.GetStarredIds(authedEmail, year)
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

func (s *Server) registerUserRoutes(group *gin.RouterGroup) {
	group.GET("/user", s.GetUser)
	group.GET("/user/events/:email/:year", s.LoadUserEvents)
}
