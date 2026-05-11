package background

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/logging"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func parseHttp(sourceFile string) []*events.GenconEvent {
	resp, err := http.Get(sourceFile)
	
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	spreadsheetBytes, err := ioutil.ReadAll(resp.Body)
	return events.ParseGenconSheet(spreadsheetBytes)
}

func parseSheet(sourceFile string) []*events.GenconEvent {
	fileReader, err := os.Open(sourceFile)

	if err != nil {
		panic(err)
	}
	defer fileReader.Close()
	fileBytes, err := ioutil.ReadAll(fileReader)

	return events.ParseGenconSheet(fileBytes)
}

func parseCsv(sourceFile string) []*events.GenconEvent {
	fileReader, err := os.Open(sourceFile)

	if err != nil {
		panic(err)
	}
	defer fileReader.Close()
	fileBytes, err := ioutil.ReadAll(fileReader)

	return events.ParseGenconCsv(fileBytes)
}

func writeEvents(db *sql.DB, genconEvents []*events.GenconEvent) (postgres.UpdateStats, error) {
	var stats postgres.UpdateStats
	tx, err := db.Begin()
	if err != nil {
		logging.LogWithError(err, "Error beginning transaction")
		return stats, err
	}
	stats, err = postgres.BulkUpdateEvents(tx, genconEvents)
	if err != nil {
		logging.LogWithError(err, "Error bulk updating events")
		tx.Rollback()
		return stats, err
	}
	err = tx.Commit()
	if err != nil {
		logging.LogWithError(err, "Error committing transaction")
		return stats, err
	}
	return stats, nil
}

func UpdateEventsFromGencon(db *sql.DB, sourceFile string) {
	var events []*events.GenconEvent
	log.Printf("Loading events from %v", sourceFile)

	var stats postgres.UpdateStats
	var err error

	defer func() {
		// Recover any panic, log error, and save failure state before re-panicking
		if r := recover(); r != nil {
			err = r.(error)
			postgres.LogUpdate(db, stats, false, err.Error())
			panic(r)
		} else if err != nil {
			postgres.LogUpdate(db, stats, false, err.Error())
		} else {
			postgres.LogUpdate(db, stats, true, "")
		}
	}()

	if strings.HasPrefix(sourceFile, "http") {
		events = parseHttp(sourceFile)
	} else if strings.HasSuffix(sourceFile, "xlsx") {
		events = parseSheet(sourceFile)
	} else {
		events = parseCsv(sourceFile)
	}

	stats, err = writeEvents(db, events)
	if err != nil {
		log.Fatal(err)
	}
}
