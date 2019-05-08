package web

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
		defaultYear := time.Now().Year()

		var err error
		if len(strings.TrimSpace(c.Param("year"))) > 0 {
			appContext.Year, err = strconv.Atoi(c.Param("year"))
			if err != nil {
				log.Printf("Error parsing year")
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		} else {
			appContext.Year = defaultYear
		}
		c.HTML(http.StatusOK, "starred.html", gin.H{
			"context": appContext,
		})
	}
}
