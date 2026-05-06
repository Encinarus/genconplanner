package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"time"
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

type StarEventRequest struct {
	EventId string `json:"eventId"`
	Add     bool   `json:"add"`
	Related bool   `json:"related"`
}

func (s *Server) StarEvent(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req StarEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	starred, err := s.Repo.UpdateStarredEvent(email, req.EventId, req.Related, req.Add)
	if err != nil {
		log.Printf("error updating starred event: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, starred)
}

func (s *Server) GetStarredEvents(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	// This is a bit inefficient as it loads GenconEvents then converts,
	// but it reuses existing postgres logic.
	log.Printf("Loading starred events for %s year %d\n", email, year)
	dbEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		log.Printf("error loading starred events: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	log.Printf("Found %d starred events\n", len(dbEvents))

	apiResults := make([]EventSummary, 0)
	for _, dbEvent := range dbEvents {
		var apiEvent Event
		convertEvent(&apiEvent, dbEvent)
		
		// Convert to summary for the list view
		var summary EventSummary
		summary.AnchorEventId = apiEvent.EventId
		summary.Title = apiEvent.Title
		summary.ShortDescription = apiEvent.ShortDescription
		summary.GameSystem = s.lookupGame(dbEvent.GameSystem)
		
		switch dbEvent.StartTime.Weekday() {
		case time.Wednesday:
			summary.WedTickets = dbEvent.TicketsAvailable
		case time.Thursday:
			summary.ThuTickets = dbEvent.TicketsAvailable
		case time.Friday:
			summary.FriTickets = dbEvent.TicketsAvailable
		case time.Saturday:
			summary.SatTickets = dbEvent.TicketsAvailable
		case time.Sunday:
			summary.SunTickets = dbEvent.TicketsAvailable
		}
		
		apiResults = append(apiResults, summary)
	}

	c.JSON(http.StatusOK, apiResults)
}

func (s *Server) registerUserRoutes(group *gin.RouterGroup) {
	group.GET("/user", s.GetUser)
	group.GET("/user/events/:email/:year", s.LoadUserEvents)
	group.GET("/user/starred/:year", s.GetStarredEvents)
	group.POST("/user/star", s.StarEvent)
}
