package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/Encinarus/genconplanner/internal/events"
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

type StarredEventDetail struct {
	EventId          string `json:"eventId"`
	Title            string `json:"title"`
	ShortDescription string `json:"shortDescription"`
	CategoryCode     string `json:"categoryCode"`
	StartTime        string `json:"startTime"`
	EndTime          string `json:"endTime"`
	GenconUrl        string `json:"genconUrl"`
	PlannerUrl       string `json:"plannerUrl"`
}

type StarredPageData struct {
	Email            string                 `json:"email"`
	Year             int                    `json:"year"`
	CalendarEvents   []CalendarEventCluster `json:"calendarEvents"`
	IndividualEvents []StarredEventDetail   `json:"individualEvents"`
	Metadata         CalendarMetadata       `json:"metadata"`
	StarredClusters  []string               `json:"starredClusters"`
	StarredEvents    []string               `json:"starredEvents"`
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

	starred, err := s.Repo.UpdateStarredEventMinimal(email, req.EventId, req.Related, req.Add)
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

	// Load event groups (clusters) to match legacy UI behavior
	log.Printf("Loading starred event groups for %s year %d\n", email, year)
	dbGroups, err := s.Repo.LoadStarredEventGroups(email, year)
	if err != nil {
		log.Printf("error loading starred event groups: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	log.Printf("Found %d starred event groups\n", len(dbGroups))

	apiResults := make([]EventSummary, 0)
	for _, dbGroup := range dbGroups {
		var summary EventSummary
		summary.AnchorEventId = dbGroup.EventId
		summary.Title = dbGroup.Name
		summary.ShortDescription = dbGroup.Description
		summary.NumEvents = dbGroup.Count
		summary.GameSystem.Name = dbGroup.GameSystem
		
		summary.WedTickets = dbGroup.WedTickets
		summary.ThuTickets = dbGroup.ThursTickets
		summary.FriTickets = dbGroup.FriTickets
		summary.SatTickets = dbGroup.SatTickets
		summary.SunTickets = dbGroup.SunTickets
		
		apiResults = append(apiResults, summary)
	}

	c.JSON(http.StatusOK, apiResults)
}

func (s *Server) GetStarredCalendarEvents(c *gin.Context) {
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

	starredEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	clusters, err := s.Repo.LoadStarredEventClusters(email, year, starredEvents)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	apiClusters := make([]CalendarEventCluster, 0, len(clusters))
	for _, cluster := range clusters {
		apiClusters = append(apiClusters, CalendarEventCluster{
			Title:            cluster.Title,
			StartTime:        cluster.StartTime,
			EndTime:          cluster.EndTime,
			GenconUrl:        cluster.GenconUrl,
			PlannerUrl:       cluster.PlannerUrl,
			ShortCategory:    cluster.ShortCategory,
			ShortDescription: cluster.ShortDescription,
			SimilarCount:     cluster.SimilarCount,
		})
	}

	c.JSON(http.StatusOK, apiClusters)
}

func (s *Server) GetCalendarMetadata(c *gin.Context) {
	yearParam := c.Param("year")
	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	c.JSON(http.StatusOK, CalendarMetadata{
		StartDate: events.GenconStartDate(year),
		EndDate:   events.GenconEndDate(year),
	})
}

func (s *Server) GetStarredIndividualEvents(c *gin.Context) {
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

	dbEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		log.Printf("error loading starred individual events: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	results := make([]StarredEventDetail, 0, len(dbEvents))
	for _, e := range dbEvents {
		results = append(results, StarredEventDetail{
			EventId:          e.EventId,
			Title:            e.Title,
			ShortDescription: e.ShortDescription,
			CategoryCode:     e.ShortCategory,
			StartTime:        e.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			EndTime:          e.EndTime.Format("2006-01-02T15:04:05Z07:00"),
			GenconUrl:        e.GenconLink(),
			PlannerUrl:       e.PlannerLink(),
		})
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) GetStarredPageData(c *gin.Context) {
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

	// 1. Get starred events (needed for both clusters and details)
	dbEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		log.Printf("error loading starred events: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// 2. Get clusters for calendar
	clusters, err := s.Repo.LoadStarredEventClusters(email, year, dbEvents)
	if err != nil {
		log.Printf("error loading clusters: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// 3. Get Starred IDs for global state sync
	starredIds, err := s.Repo.GetStarredIds(email, year)
	if err != nil {
		log.Printf("error getting starred ids: %v\n", err)
	}

	var data StarredPageData
	data.Email = email
	data.Year = year
	data.Metadata = CalendarMetadata{
		StartDate: events.GenconStartDate(year),
		EndDate:   events.GenconEndDate(year),
	}

	for _, e := range dbEvents {
		data.IndividualEvents = append(data.IndividualEvents, StarredEventDetail{
			EventId:          e.EventId,
			Title:            e.Title,
			ShortDescription: e.ShortDescription,
			CategoryCode:     e.ShortCategory,
			StartTime:        e.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			EndTime:          e.EndTime.Format("2006-01-02T15:04:05Z07:00"),
			GenconUrl:        e.GenconLink(),
			PlannerUrl:       e.PlannerLink(),
		})
	}

	for _, cluster := range clusters {
		data.CalendarEvents = append(data.CalendarEvents, CalendarEventCluster{
			Title:            cluster.Title,
			StartTime:        cluster.StartTime,
			EndTime:          cluster.EndTime,
			GenconUrl:        cluster.GenconUrl,
			PlannerUrl:       cluster.PlannerUrl,
			ShortCategory:    cluster.ShortCategory,
			ShortDescription: cluster.ShortDescription,
			SimilarCount:     cluster.SimilarCount,
		})
	}

	if starredIds != nil {
		for _, s := range starredIds.StarredEvents {
			if s.Level == "group" {
				data.StarredClusters = append(data.StarredClusters, s.EventId)
			} else {
				data.StarredEvents = append(data.StarredEvents, s.EventId)
			}
		}
	}

	c.JSON(http.StatusOK, data)
}

func (s *Server) registerUserRoutes(group *gin.RouterGroup) {
	group.GET("/user", s.GetUser)
	group.GET("/user/events/:email/:year", s.LoadUserEvents)
	group.GET("/user/starred/:year", s.GetStarredEvents)
	group.GET("/user/starred/list/:year", s.GetStarredIndividualEvents)
	group.GET("/user/starred/calendar/:year", s.GetStarredCalendarEvents)
	group.GET("/user/starred/page/:year", s.GetStarredPageData)
	group.GET("/calendar/metadata/:year", s.GetCalendarMetadata)
	group.POST("/user/star", s.StarEvent)
}
