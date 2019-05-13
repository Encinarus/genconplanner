package web

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func GetStarredEvents(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		appContext := c.MustGet("context").(*Context)

		if appContext.Email == "" {
			c.JSON(http.StatusOK, &postgres.UserStarredEvents{})
			return
		}

		starredRows, err := postgres.GetStarredIds(db, appContext.Email)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.JSON(http.StatusOK, starredRows)
	}
}

func StarEvent(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		appContext := c.MustGet("context").(*Context)

		if appContext.Email == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		eventId := c.PostForm("eventId")
		related, err := strconv.ParseBool(c.PostForm("related"))
		if err != nil {
			related = false
		}
		add, err := strconv.ParseBool(c.PostForm("add"))
		if err != nil {
			add = false
		}

		log.Printf("Updating starred: %v, %v, %v\n", eventId, related, add)

		starredRows, err := postgres.UpdateStarredEvent(db, appContext.Email, eventId, related, add)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, starredRows)
	}
}

func StarredPage(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		appContext := c.MustGet("context").(*Context)
		appContext.Year = time.Now().Year()

		var err error
		if len(strings.TrimSpace(c.Param("year"))) > 0 {
			appContext.Year, err = strconv.Atoi(c.Param("year"))
			if err != nil {
				log.Printf("Error parsing year")
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		}

		starredEvents, err := postgres.LoadStarredEvents(db, appContext.Email, appContext.Year)
		if err != nil {
			log.Printf("Error loading starred events")
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		groupedEvents, err := postgres.LoadStarredEventClusters(db, appContext.Email, appContext.Year, starredEvents)
		if err != nil {
			log.Printf("Error loading starred groups")
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		log.Printf("Loaded calendar groups: %v\n", groupedEvents)
		c.Header("Cache-Control", "no-cache")
		c.HTML(http.StatusOK, "starred.html", gin.H{
			"context":          appContext,
			"eventsByDay":      events.PartitionEventsByDay(starredEvents),
			"eventsByCategory": events.PartitionEventsByCategory(starredEvents),
			"allCategories":    events.AllCategories(),
			"calendarGroups":   groupedEvents,
		})
	}
}
