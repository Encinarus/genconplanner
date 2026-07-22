package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

// Search param struct for looking up events
type EventsSearch struct {
	Category      string `form:"cat"`
	Year          int    `form:"year"`
	MinWedTickets int    `form:"minWedTickets"`
	MinThuTickets int    `form:"minThuTickets"`
	MinFriTickets int    `form:"minFriTickets"`
	MinSatTickets int    `form:"minSatTickets"`
	MinSunTickets int    `form:"minSunTickets"`

	TextQuery string `form:"search"`
	OrgId     int    `form:"org_id"`
	OnlyFree  bool   `form:"free"`
}

// Used in search results
type EventSummary struct {
	AnchorEventId    string     `json:"anchorEventId"`
	Title            string     `json:"title"`
	ShortDescription string     `json:"shortDescription"`
	NumEvents        int        `json:"numEvents"`
	WedEvents        int        `json:"wedEvents"`
	WedTotalTickets  int        `json:"wedTotalTickets"`
	WedTickets       int        `json:"wedTickets"`
	ThuEvents        int        `json:"thuEvents"`
	ThuTotalTickets  int        `json:"thuTotalTickets"`
	ThuTickets       int        `json:"thuTickets"`
	FriEvents        int        `json:"friEvents"`
	FriTotalTickets  int        `json:"friTotalTickets"`
	FriTickets       int        `json:"friTickets"`
	SatEvents        int        `json:"satEvents"`
	SatTotalTickets  int        `json:"satTotalTickets"`
	SatTickets       int        `json:"satTickets"`
	SunEvents        int        `json:"sunEvents"`
	SunTotalTickets  int        `json:"sunTotalTickets"`
	SunTickets       int        `json:"sunTickets"`
	OrgId            int        `json:"orgId"`
	CategoryCode     string     `json:"categoryCode"`
	GameSystem       GameSystem `json:"gameSystem"`
}

type GameSystem struct {
	Name          string  `json:"name"`
	BggId         int64   `json:"bggId,omitempty"`
	BggRating     float64 `json:"bggRating,omitempty"`
	NumBggRatings int64   `json:"numBggRatings,omitempty"`
	YearPublished int64   `json:"yearPublished,omitempty"`
}

type EventRef struct {
	EventId          string    `json:"eventId"`
	TicketsAvailable int       `json:"ticketsAvailable"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
}

type CalendarEventCluster struct {
	EventId          string    `json:"eventId"`
	Title            string    `json:"title"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	GenconUrl        string    `json:"genconUrl"`
	PlannerUrl       string    `json:"plannerUrl"`
	ShortCategory    string    `json:"shortCategory"`
	ShortDescription string    `json:"shortDescription"`
	SimilarCount     int       `json:"similarCount"`
	Location         string    `json:"location"`
	RoomName         string    `json:"roomName"`
	TableNumber      string    `json:"tableNumber"`
	PartyMembers     []string  `json:"partyMembers"`
}

type CalendarMetadata struct {
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

type Event struct {
	EventId              string     `json:"eventId"`
	Year                 int        `json:"year"`
	Active               bool       `json:"active"`
	Title                string     `json:"title"`
	ShortDescription     string     `json:"shortDescription"`
	LongDescription      string     `json:"longDescription"`
	CategoryCode         string     `json:"categoryCode"`
	EventType            string     `json:"eventType"`
	Group                string     `json:"group"`
	OrgId                int64      `json:"orgId"`
	GameSystem           GameSystem `json:"gameSystem"`
	RulesEdition         string     `json:"rulesEdition"`
	MinPlayers           int        `json:"minPlayers"`
	MaxPlayers           int        `json:"maxPlayers"`
	AgeRequired          string     `json:"ageRequired"`
	ExperienceRequired   string     `json:"experienceRequired"`
	MaterialsProvided    bool       `json:"materialsProvided"`
	StartTime            time.Time  `json:"startTime"`
	Duration             int        `json:"duration"`
	EndTime              time.Time  `json:"endTime"`
	GMNames              string     `json:"gmNames"`
	Website              string     `json:"website"`
	Email                string     `json:"email"`
	IsTournament         bool       `json:"isTournament"`
	RoundNumber          int        `json:"roundNumber"`
	TotalRounds          int        `json:"totalRounds"`
	MinPlayTime          int        `json:"minPlayTime"`
	AttendeeRegistration string     `json:"attendeeRegistration"`
	Cost                 int        `json:"cost"`
	Location             string     `json:"location"`
	RoomName             string     `json:"roomName"`
	TableNumber          string     `json:"tableNumber"`
	TicketsAvailable     int        `json:"ticketsAvailable"`
	LastModified         time.Time  `json:"lastModified"`
	RelatedEvents        []EventRef `json:"relatedEvents"`
	GenconUrl            string     `json:"genconUrl"`
}

func convertEvent(apiEvent *Event, dbEvent *events.GenconEvent) {
	apiEvent.EventId = dbEvent.EventId
	apiEvent.Year = dbEvent.Year
	apiEvent.Active = dbEvent.Active
	apiEvent.Title = dbEvent.Title
	apiEvent.ShortDescription = dbEvent.ShortDescription
	apiEvent.LongDescription = dbEvent.LongDescription

	apiEvent.CategoryCode = dbEvent.ShortCategory
	apiEvent.EventType = dbEvent.EventType
	apiEvent.Group = dbEvent.Group
	apiEvent.OrgId = dbEvent.OrgId
	// apiEvent.GameSystem is handled elsewhere
	apiEvent.RulesEdition = dbEvent.RulesEdition
	apiEvent.MinPlayers = dbEvent.MinPlayers
	apiEvent.MaxPlayers = dbEvent.MaxPlayers
	apiEvent.AgeRequired = dbEvent.AgeRequired
	apiEvent.ExperienceRequired = dbEvent.ExperienceRequired
	apiEvent.MaterialsProvided = dbEvent.MaterialsProvided
	apiEvent.StartTime = dbEvent.StartTime
	apiEvent.Duration = dbEvent.Duration
	apiEvent.EndTime = dbEvent.EndTime
	apiEvent.GMNames = dbEvent.GMNames
	apiEvent.Website = dbEvent.Website
	apiEvent.Email = dbEvent.Email
	apiEvent.IsTournament = dbEvent.Tournament
	apiEvent.RoundNumber = dbEvent.RoundNumber
	apiEvent.TotalRounds = dbEvent.TotalRounds
	apiEvent.MinPlayTime = dbEvent.MinPlayTime
	apiEvent.AttendeeRegistration = dbEvent.AttendeeRegistration
	apiEvent.Cost = dbEvent.Cost
	apiEvent.Location = dbEvent.Location
	apiEvent.RoomName = dbEvent.RoomName
	apiEvent.TableNumber = dbEvent.TableNumber
	apiEvent.TicketsAvailable = dbEvent.TicketsAvailable
	apiEvent.LastModified = dbEvent.LastModified
	apiEvent.GenconUrl = dbEvent.GenconLink()
}

func (s *Server) lookupGame(gameSystem string) GameSystem {
	result := GameSystem{Name: gameSystem}

	dbGame := s.Games.FindGame(gameSystem)
	if dbGame != nil {
		result.BggId = dbGame.BggId
		result.BggRating = dbGame.AvgRatings
		result.NumBggRatings = dbGame.NumRatings
		result.YearPublished = dbGame.YearPublished
	}

	return result
}

func (s *Server) LookupEvent(c *gin.Context) {
	eventId := c.Param("event_id")
	if len(strings.TrimSpace(eventId)) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Missing event_id"})
		return
	}

	var apiEvent Event
	dbEvents, err := s.Repo.LoadSimilarEvents(c.Request.Context(), eventId, "")

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	if len(dbEvents) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Event not found"})
		return
	}

	for i := range dbEvents {
		dbEvent := dbEvents[i]

		if dbEvent.EventId == eventId {
			convertEvent(&apiEvent, dbEvent)
			apiEvent.GameSystem = s.lookupGame(dbEvent.GameSystem)
		}

		// Add all events (including the current one) to RelatedEvents
		var related EventRef
		related.EventId = dbEvent.EventId
		related.StartTime = dbEvent.StartTime
		related.EndTime = dbEvent.EndTime
		related.TicketsAvailable = dbEvent.TicketsAvailable
		apiEvent.RelatedEvents = append(apiEvent.RelatedEvents, related)
	}

	c.JSON(http.StatusOK, apiEvent)
}

func convertEventGroup(dbEventGroup *postgres.EventGroup) *EventSummary {
	if dbEventGroup == nil {
		return nil
	}
	var apiEventSummary EventSummary
	apiEventSummary.AnchorEventId = dbEventGroup.EventId
	apiEventSummary.Title = dbEventGroup.Name
	apiEventSummary.ShortDescription = dbEventGroup.Description
	apiEventSummary.NumEvents = dbEventGroup.Count
	apiEventSummary.WedEvents = dbEventGroup.WedEvents
	apiEventSummary.WedTotalTickets = dbEventGroup.WedTotalTickets
	apiEventSummary.WedTickets = dbEventGroup.WedTickets
	apiEventSummary.ThuEvents = dbEventGroup.ThursEvents
	apiEventSummary.ThuTotalTickets = dbEventGroup.ThursTotalTickets
	apiEventSummary.ThuTickets = dbEventGroup.ThursTickets
	apiEventSummary.FriEvents = dbEventGroup.FriEvents
	apiEventSummary.FriTotalTickets = dbEventGroup.FriTotalTickets
	apiEventSummary.FriTickets = dbEventGroup.FriTickets
	apiEventSummary.SatEvents = dbEventGroup.SatEvents
	apiEventSummary.SatTotalTickets = dbEventGroup.SatTotalTickets
	apiEventSummary.SatTickets = dbEventGroup.SatTickets
	apiEventSummary.SunEvents = dbEventGroup.SunEvents
	apiEventSummary.SunTotalTickets = dbEventGroup.SunTotalTickets
	apiEventSummary.SunTickets = dbEventGroup.SunTickets
	apiEventSummary.OrgId = dbEventGroup.OrgId
	apiEventSummary.CategoryCode = dbEventGroup.ShortCategory

	return &apiEventSummary
}

func (s *Server) SearchEvents(c *gin.Context) {
	var search EventsSearch

	err := c.ShouldBind(&search)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid search parameters"})
		return
	}

	if search.Year == 0 {
		// Default to this year if not specified.
		search.Year = time.Now().Year()
	}

	var q postgres.SearchQuery
	q.CategoryShortCode = search.Category
	q.Year = search.Year
	q.RawQuery = search.TextQuery
	q.MinWedTickets = search.MinWedTickets
	q.MinThuTickets = search.MinThuTickets
	q.MinFriTickets = search.MinFriTickets
	q.MinSatTickets = search.MinSatTickets
	q.MinSunTickets = search.MinSunTickets
	q.OrgId = search.OrgId
	q.OnlyFree = search.OnlyFree
	q.UserEmail = GetUserEmail(c)

	var matches []*postgres.EventGroup

	if len(strings.TrimSpace(search.TextQuery)) == 0 && len(search.Category) > 0 && !search.OnlyFree {
		matches, err = s.Repo.LoadEventGroupsForCategory(c.Request.Context(), search.Category, search.Year)
	} else {
		matches, err = s.Repo.SearchEvents(c.Request.Context(), q)
	}

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	apiResults := make([]EventSummary, 0)
	for _, match := range matches {
		eventGroup := convertEventGroup(match)
		if eventGroup == nil {
			continue
		}
		eventGroup.GameSystem = s.lookupGame(match.GameSystem)
		apiResults = append(apiResults, *eventGroup)
	}

	c.JSON(http.StatusOK, apiResults)
}

func (s *Server) registerEventRoutes(group *gin.RouterGroup) {
	group.GET("/event/:event_id", s.LookupEvent)
	group.GET("/events", s.OptionalAuth(), s.SearchEvents)
	group.POST("/events", s.OptionalAuth(), s.SearchEvents)
}
