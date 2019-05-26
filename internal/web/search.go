package web

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseQuery(searchQuery string, year int, days map[string]bool) *postgres.ParsedQuery {
	query := postgres.ParsedQuery{
		Year:       year,
		DaysOfWeek: days,
		RawQuery:   searchQuery,
	}

	maxYear := time.Now().Year()
	if year > maxYear || year < 2016 {
		query.Year = maxYear
	}

	// Preprocess, removing symbols which are used in tsquery
	searchQuery = strings.Replace(searchQuery, "!", "", -1)
	searchQuery = strings.Replace(searchQuery, "&", "", -1)
	searchQuery = strings.Replace(searchQuery, "(", "", -1)
	searchQuery = strings.Replace(searchQuery, ")", "", -1)
	searchQuery = strings.Replace(searchQuery, "|", "", -1)

	queryReader := csv.NewReader(bytes.NewBufferString(searchQuery))
	queryReader.Comma = ' '

	splitQuery, _ := queryReader.Read()

	for _, term := range splitQuery {
		invertTerm := false
		if strings.HasPrefix(term, "-") {
			term = strings.TrimLeft(term, "-")
			invertTerm = true
		}
		if strings.ContainsAny(term, ":<>=-~") {
			// TODO(alek) Handle key:value searches
			// : and = work as equals
			// < > compare for dates or num tickets
			// ~ is for checking if the string is in a field
			continue
		}

		// Now remove remaining symbols we want to allow in field-specific
		// searches, but not in the general text search
		term = strings.Replace(term, "<", "", -1)
		term = strings.Replace(term, ">", "", -1)
		term = strings.Replace(term, "=", "", -1)
		term = strings.Replace(term, "-", "", -1)
		term = strings.Replace(term, "~", "", -1)
		term = strings.TrimSpace(term)
		if len(term) == 0 {
			continue
		}
		if invertTerm {
			term = "!" + term
		}
		query.TextQueries = append(query.TextQueries, term)
	}
	query.DaysOfWeek = days
	return &query
}

func parseHour(c *gin.Context, param string, defaultValue int) int {
	raw, found := c.GetQuery(param)
	if !found {
		return defaultValue
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	} else if parsed < 0 || parsed > 24 {
		return defaultValue
	} else {
		return parsed
	}
}

func Search(db *sql.DB) func(c *gin.Context) {
	keyFunc := func(g *postgres.EventGroup) (string, string) {
		majorGroup := events.LongCategory(g.ShortCategory)
		minorGroup := "Unspecified"

		if len(strings.TrimSpace(g.GameSystem)) != 0 {
			minorGroup = strings.TrimSpace(g.GameSystem)
		}

		return majorGroup, minorGroup
	}

	return func(c *gin.Context) {
		query := c.Query("q")
		year, err := strconv.Atoi(c.Query("y"))
		if err != nil {
			year = time.Now().Year()
		}

		days := make(map[string]bool)
		for _, day := range []string{"wed", "thu", "fri", "sat", "sun"} {
			param, found := c.GetQuery(day)

			if found && len(param) > 0 {
				if b, err := strconv.ParseBool(param); err == nil {
					days[day] = b
				}
			}
		}
		parsedQuery := parseQuery(query, year, days)

		parsedQuery.StartBeforeHour = parseHour(c, "start_before", 24)
		parsedQuery.StartAfterHour = parseHour(c, "start_after", 0)
		parsedQuery.EndBeforeHour = parseHour(c, "end_before", 24)
		parsedQuery.EndAfterHour = parseHour(c, "end_after", 0)

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
			appContext.Year = year

			majorHeadings, minorHeadings, partitions := PartitionGroups(eventGroups, keyFunc)
			c.HTML(http.StatusOK, "results.html", gin.H{
				"context":       appContext,
				"majorHeadings": majorHeadings,
				"minorHeadings": minorHeadings,
				"partitions":    partitions,
				"totalEvents":   totalEvents,
				"groups":        len(eventGroups),
				"breakdown":     "Category",
				"pageHeader":    "Search",
				"subHeader":     query,
				"query":         parsedQuery,
			})
		}
	}
}
