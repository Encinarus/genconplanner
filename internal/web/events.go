package web

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type LookupResult struct {
	MainEvent    *events.GenconEvent
	EventsPerDay map[string][]*events.GenconEvent
	TotalTickets int
}

func lookupEvent(db *sql.DB, eventId string, userEmail string) (*LookupResult, error) {
	foundEvents, err := postgres.LoadSimilarEvents(db, eventId, userEmail)
	if err != nil {
		return nil, err
	}
	log.Printf("Found %v events similar to %s", len(foundEvents), eventId)

	result := LookupResult{
		EventsPerDay: events.PartitionEventsByDay(foundEvents),
	}
	for _, event := range foundEvents {
		if event.EventId == eventId {
			result.MainEvent = event
		}

		result.TotalTickets += event.TicketsAvailable
	}

	return &result, nil
}

func allStarred(events []*events.GenconEvent) bool {
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
		result, err := lookupEvent(db, eventId, appContext.Email)
		if err != nil {
			log.Printf("Unable to lookup event %v\n", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		appContext.Year = result.MainEvent.Year

		starred := true
		for _, loadedEvents := range result.EventsPerDay {
			starred = starred && allStarred(loadedEvents)
		}
		c.HTML(http.StatusOK, "event.html", gin.H{
			"result":       result,
			"eventsPerDay": result.EventsPerDay,
			"context":      appContext,
			"allStarred":   starred,
		})
	}
}
