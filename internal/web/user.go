package web

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"time"
)

func User(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		appContext := c.MustGet("context").(*Context)
		year, err := strconv.Atoi(c.Param("year"))
		if err != nil {
			year = time.Now().Year()
		}

		appContext.Year = year
		if appContext.Email == "" {
			c.HTML(http.StatusUnauthorized, "signin.html", gin.H{
				"context":  appContext,
				"redirect": c.Request.URL,
			})
			return
		}

		parties, err := postgres.LoadParties(db, appContext.User)
		if err != nil {
			log.Printf("Unable to load parties: %v", err)
		} else {
			log.Printf("Num parties: %v", len(parties))
		}

		c.HTML(http.StatusOK, "user.html", gin.H{
			"context": appContext,
			"user":    appContext.User,
			"parties": parties,
		})
	}
}

func UserNameChange(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		year, err := strconv.Atoi(c.Param("year"))
		if err != nil {
			year = time.Now().Year()
		}
		appContext := c.MustGet("context").(*Context)
		appContext.Year = year

		c.HTML(http.StatusOK, "user.html", gin.H{
			"context": appContext,
			"user":    appContext.User,
		})
	}
}
