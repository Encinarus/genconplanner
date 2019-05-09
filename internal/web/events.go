package web

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

type LookupResult struct {
	MainEvent    *events.GenconEvent
	Wednesday    []*events.SlimEvent
	Thursday     []*events.SlimEvent
	Friday       []*events.SlimEvent
	Saturday     []*events.SlimEvent
	Sunday       []*events.SlimEvent
	TotalTickets int
}

func lookupEvent(db *sql.DB, eventId string, userEmail string) *LookupResult {
	foundEvents, err := postgres.LoadSimilarEvents(db, eventId, userEmail)
	if err != nil {
		log.Fatalf("Unable to load events, err %v", err)
	}
	log.Printf("Found %v events similar to %s", len(foundEvents), eventId)

	var result LookupResult
	for _, event := range foundEvents {
		if event.EventId == eventId {
			result.MainEvent = event
		}

		switch event.StartTime.Weekday() {
		case time.Wednesday:
			result.Wednesday = append(result.Wednesday, event.SlimEvent())
			break
		case time.Thursday:
			result.Thursday = append(result.Thursday, event.SlimEvent())
			break
		case time.Friday:
			result.Friday = append(result.Friday, event.SlimEvent())
			break
		case time.Saturday:
			result.Saturday = append(result.Saturday, event.SlimEvent())
			break
		case time.Sunday:
			result.Sunday = append(result.Sunday, event.SlimEvent())
			break
		}

		result.TotalTickets += event.TicketsAvailable
	}

	return &result
}

func allStarred(events []*events.SlimEvent) bool {
	for _, similar := range events {
		if !similar.IsStarred {
			return false
		}
	}
	return true
}

func ViewEvent(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId := c.Param("eid")
		appContext := c.MustGet("context").(*Context)
		result := lookupEvent(db, eventId, appContext.Email)
		appContext.Year = result.MainEvent.Year

		starred := allStarred(result.Wednesday)
		starred = starred && allStarred(result.Thursday)
		starred = starred && allStarred(result.Friday)
		starred = starred && allStarred(result.Saturday)
		starred = starred && allStarred(result.Sunday)

		c.HTML(http.StatusOK, "event.html", gin.H{
			"result":     result,
			"context":    appContext,
			"allStarred": starred,
		})
	}
}
