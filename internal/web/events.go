package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
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

func renderHtml(c *gin.Context, result *LookupResult, appContext *Context) {
	starred := true
	for _, loadedEvents := range result.EventsPerDay {
		starred = starred && allStarred(loadedEvents)
	}
	nextUpdateTime := time.Now().Add(time.Hour).Truncate(time.Hour).Add(time.Minute * 5)
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", nextUpdateTime.Sub(time.Now())/time.Second))

	c.HTML(http.StatusOK, "event.html", gin.H{
		"result":       result,
		"eventsPerDay": result.EventsPerDay,
		"context":      appContext,
		"allStarred":   starred,
	})
}

func renderJson(c *gin.Context, result *LookupResult, appContext *Context) {
	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.Writer).Encode(result)
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

		_, json := c.GetQuery("json")
		if json {
			renderJson(c, result, appContext)
		} else {
			renderHtml(c, result, appContext)
		}
	}
}
