package web

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseQuery(searchQuery string, year int, rawDays string) *postgres.ParsedQuery {
	query := postgres.ParsedQuery{}

	query.Year = time.Now().Year()

	if year <= query.Year && year > 2016 {
		query.Year = year
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

	// TODO(alek): consider adding a db field "searchable_text" rather than relying
	// the trigger across many fields. Then exact matches do like vs that, while fuzzy
	// matches go against the ts_vector column
	for _, term := range splitQuery {
		invertTerm := false
		if strings.HasPrefix(term, "-") {
			log.Println("Negated term:", term)
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
	query.DaysOfWeek = ParseDayQuery(rawDays)
	return &query
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
		log.Println("Raw Query: ", query)
		days := c.Query("days")

		parsedQuery := parseQuery(query, year, days)

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
			})
		}
	}
}
