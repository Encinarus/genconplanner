package web

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

func CategoryList(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		defaultYear := time.Now().Year()

		var err error
		context := c.MustGet("context").(*Context)

		if len(strings.TrimSpace(c.Param("year"))) > 0 {
			context.Year, err = strconv.Atoi(c.Param("year"))
			if err != nil {
				log.Printf("Error parsing year")
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		} else {
			context.Year = defaultYear
		}

		summary, err := postgres.LoadCategorySummary(db, context.Year)

		if err != nil {
			log.Printf("Error loading categories, %v", err)
			c.AbortWithError(500, err)
			return
		}

		batchSize := 2
		tail := len(summary) % batchSize
		numBuckets := len(summary) / batchSize
		if tail != 0 {
			numBuckets++
		}
		categories := make([][]*postgres.CategorySummary, numBuckets)
		for i := range categories {
			base := batchSize * i
			end := base + batchSize
			if end > len(summary) {
				end = len(summary)
			}
			categories[i] = summary[base:end]
		}

		c.HTML(http.StatusOK, "categories.html", gin.H{
			"title":      "Main website",
			"categories": categories,
			"context":    context,
		})
	}
}

func ViewCategory(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		params := processQueryParams(c)

		appContext := c.MustGet("context").(*Context)
		appContext.Year = params.Year

		if len(params.Category) == 0 {
			log.Printf("No category specified")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		eventGroups, err := postgres.LoadEventGroups(db, params.Category, params.Year, []int{})
		if err != nil {
			log.Printf("Error loading event groups")
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		totalEvents := 0
		for _, group := range eventGroups {
			totalEvents += group.Count
		}
		majorHeadings, minorHeadings, partitions := PartitionGroups(eventGroups, appContext, params)
		c.HTML(http.StatusOK, "results.html", gin.H{
			"context":       appContext,
			"majorHeadings": majorHeadings,
			"minorHeadings": minorHeadings,
			"partitions":    partitions,
			"totalEvents":   totalEvents,
			"groups":        len(eventGroups),
			"breakdown":     "Category",
			"pageHeader":    "Search",
			"subHeader":     params.Category,
		})
	}
}
