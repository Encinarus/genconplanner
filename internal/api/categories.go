package api

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

type Category struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	EventCount int    `json:"eventCount"`
	Year       int    `json:"year"`
}

func listCategories(c *gin.Context, db *sql.DB) {
	year := 0
	var err error
	if len(strings.TrimSpace(c.Param("year"))) > 0 {
		year, err = strconv.Atoi(c.Param("year"))
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
	}

	if year < 2020 {
		c.AbortWithStatus(400)
		return
	}

	summary, err := postgres.LoadCategorySummary(db, year)

	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	results := make([]Category, 0)
	for i := range summary {
		results = append(results, Category{
			Name:       summary[i].Name,
			Code:       summary[i].Code,
			EventCount: summary[i].Count,
			Year:       year,
		})
	}

	c.Header("Content-Type", "application/json")
	json.NewEncoder(c.Writer).Encode(results)
}

func categoryRoutes(api_group *gin.RouterGroup, db *sql.DB) {
	api_group.GET("/category/:year", func(c *gin.Context) {
		listCategories(c, db)
	})
}
