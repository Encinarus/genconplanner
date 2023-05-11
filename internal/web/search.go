package web

import (
	"database/sql"
	"net/http"

	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

func Search(db *sql.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		params := processQueryParams(c)

		if params.Grouping == nil {
			params.Grouping = KeyByCategorySystem
		}

		parsedQuery := parseQuery(params)

		eventGroups, err := postgres.FindEvents(db, parsedQuery)
		totalEvents := 0
		for _, group := range eventGroups {
			totalEvents += group.Count
		}
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		} else {
			appContext := c.MustGet("context").(*Context)
			appContext.Year = params.Year

			majorHeadings, minorHeadings, partitions := PartitionGroups(eventGroups, params.Grouping)
			c.HTML(http.StatusOK, "results.html", gin.H{
				"context":       appContext,
				"majorHeadings": majorHeadings,
				"minorHeadings": minorHeadings,
				"partitions":    partitions,
				"totalEvents":   totalEvents,
				"groups":        len(eventGroups),
				"breakdown":     "Category",
				"pageHeader":    "Search",
				"subHeader":     parsedQuery.RawQuery,
				"query":         parsedQuery,
			})
		}
	}
}
