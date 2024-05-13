package api

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

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
		c.AbortWithStatus(400)
		return
	}

	var apiEvent Event
	dbEvents, err := postgres.LoadSimilarEvents(db, eventId, "")

	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if len(dbEvents) == 0 {
		c.AbortWithStatus(404)
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

func eventRoutes(api_group *gin.RouterGroup, db *sql.DB, gameCache *background.GameCache) {
	api_group.GET("/event/:event_id", func(c *gin.Context) {
		lookupEvent(c, db, gameCache)
	})
}
