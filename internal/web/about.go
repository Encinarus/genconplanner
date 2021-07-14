package web

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

func About(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		year, err := strconv.Atoi(c.Param("year"))
		if err != nil {
			year = time.Now().Year()
		}
		appContext := c.MustGet("context").(*Context)
		appContext.Year = year
		c.HTML(http.StatusOK, "about.html", gin.H{
			"context": appContext,
		})

	}
}
