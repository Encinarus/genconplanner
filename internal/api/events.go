package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Encinarus/genconplanner/internal/background"
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

	// Not yet implemented
	TextQuery string `form:"search"`
}

// Used in search results
type EventSummary struct {
	AnchorEventId    string     `json:"anchorEventId"`
	ShortDescription string     `json:"shortDescription"`
	NumEvents        int        `json:"numEvents"`
	WedTickets       int        `json:"wedTickets"`
	ThuTickets       int        `json:"thuTickets"`
	FriTickets       int        `json:"friTickets"`
	SatTickets       int        `json:"satTickets"`
	SunTickets       int        `json:"sunTickets"`
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

type Event struct {
	EventId              string     `json:"eventId"`
	Year                 int        `json:"year"`
	Active               bool       `json:"active"`
	Title                string     `json:"title"`
	ShortDescription     string     `json:"shortDescription"`
	LongDescription      string     `json:"longDescription"`
	CategoryCode         string     `json:"categoryCode"`
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
}

func convertEvent(apiEvent *Event, dbEvent *events.GenconEvent) {
	apiEvent.EventId = dbEvent.EventId
	apiEvent.Year = dbEvent.Year
	apiEvent.Active = dbEvent.Active
	apiEvent.Title = dbEvent.Title
	apiEvent.ShortDescription = dbEvent.ShortDescription
	apiEvent.LongDescription = dbEvent.LongDescription

	apiEvent.CategoryCode = dbEvent.ShortCategory
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
}

func lookupGame(gameSystem string, gameCache *background.GameCache) GameSystem {
	result := GameSystem{Name: gameSystem}

	dbGame := gameCache.FindGame(gameSystem)
	if dbGame != nil {
		result.BggId = dbGame.BggId
		result.BggRating = dbGame.AvgRatings
		result.NumBggRatings = dbGame.NumRatings
		result.YearPublished = dbGame.YearPublished
	}

	return result
}

func lookupEvent(c *gin.Context, db *sql.DB, gameCache *background.GameCache) {
	eventId := c.Param("event_id")
	if len(strings.TrimSpace(eventId)) == 0 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var apiEvent Event
	dbEvents, err := postgres.LoadSimilarEvents(db, eventId, "")

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if len(dbEvents) == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	for i := range dbEvents {
		dbEvent := dbEvents[i]

		if dbEvent.EventId == eventId {
			convertEvent(&apiEvent, dbEvent)
			apiEvent.GameSystem = lookupGame(dbEvent.GameSystem, gameCache)
		} else {
			// It's a related event
			var related EventRef
			related.EventId = dbEvent.EventId
			related.StartTime = dbEvent.StartTime
			related.EndTime = dbEvent.EndTime
			related.TicketsAvailable = dbEvent.TicketsAvailable
			apiEvent.RelatedEvents = append(apiEvent.RelatedEvents, related)
		}
	}

	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.Writer).Encode(apiEvent)
}

func convertEventGroup(dbEventGroup *postgres.EventGroup) *EventSummary {
	var apiEventSummary EventSummary
	apiEventSummary.AnchorEventId = dbEventGroup.EventId
	apiEventSummary.ShortDescription = dbEventGroup.Description
	apiEventSummary.NumEvents = dbEventGroup.Count
	apiEventSummary.WedTickets = dbEventGroup.WedTickets
	apiEventSummary.ThuTickets = dbEventGroup.ThursTickets
	apiEventSummary.FriTickets = dbEventGroup.FriTickets
	apiEventSummary.SatTickets = dbEventGroup.SatTickets
	apiEventSummary.FriTickets = dbEventGroup.FriTickets

	return &apiEventSummary
}

func searchEvents(c *gin.Context, db *sql.DB, gameCache *background.GameCache) {
	var search EventsSearch

	err := c.ShouldBind(&search)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
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

	matches, err := postgres.SearchEvents(db, q)
	// postgres.LoadEventGroupsForCategory(db, search.Category, search.Year)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	apiResults := make([]EventSummary, 0)
	for _, match := range matches {
		eventGroup := convertEventGroup(match)
		eventGroup.GameSystem = lookupGame(match.GameSystem, gameCache)
		apiResults = append(apiResults, *eventGroup)
	}

	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.Writer).Encode(apiResults)
}

func eventRoutes(api_group *gin.RouterGroup, db *sql.DB, gameCache *background.GameCache) {
	api_group.GET("/event/:event_id", func(c *gin.Context) {
		lookupEvent(c, db, gameCache)
	})

	api_group.POST("/events/", func(c *gin.Context) {
		searchEvents(c, db, gameCache)
	})
}
