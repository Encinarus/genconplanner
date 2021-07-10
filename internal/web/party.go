package web

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"time"
)

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

		c.JSON(http.StatusOK, map[string]string{
			"name":    partyName,
			"year":    strconv.FormatInt(year, 10),
			"members": appContext.Email,
		})
	}
}
