package web

import (
	"database/sql"
	"github.com/gin-gonic/gin"
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

		c.HTML(http.StatusOK, "user.html", gin.H{
			"context": appContext,
			"user":    appContext.User,
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
