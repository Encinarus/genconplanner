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

func Party(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		appContext := c.MustGet("context").(*Context)

		if appContext.Email == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var partyId int64
		var err error
		if len(strings.TrimSpace(c.Param("party_id"))) > 0 {
			partyId, err = strconv.ParseInt(c.Param("party_id"), 10, 64)
			if err != nil {
				log.Printf("Error parsing party_id")
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		} else {
			log.Printf("No party id provided")
		}

		var party *postgres.Party
		parties, err := postgres.LoadParties(db, appContext.User)
		for _, p := range parties {
			if p.Id == partyId {
				party = p
				break
			}
		}

		c.HTML(http.StatusOK, "party.html", gin.H{
			"party":       party,
			"context":      appContext,
		})
	}
}

func NewParty(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		appContext := c.MustGet("context").(*Context)

		if appContext.Email == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		log.Printf("New party via %v", c.Request.Method)
		log.Printf("Form data: %v", c.Request.Form)

		partyName := c.PostForm("partyName")
		year, err := strconv.ParseInt(c.PostForm("year"), 10, 64)
		if err != nil {
			log.Printf("Couldn't parse %v, defaulting to this year")
			year = int64(time.Now().Year())
		}
		log.Printf("Creating a new party: %v, %v, with %v as a member\n", partyName, year, appContext.Email)

		party, err := postgres.NewParty(db, partyName, year, appContext.Email)
		if err != nil {
			log.Printf("Couldn't build party: %v", err)
			year = int64(time.Now().Year())
		}

		log.Printf("Party created: %+v", party)
		c.JSON(http.StatusOK, map[string]string{
			"name":    partyName,
			"year":    strconv.FormatInt(year, 10),
			"members": appContext.Email,
		})
	}
}
