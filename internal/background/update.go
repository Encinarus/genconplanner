package background

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/events"
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

func writeEvents(db *sql.DB, genconEvents []*events.GenconEvent) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	err = postgres.BulkUpdateEvents(tx, genconEvents)
	if err != nil {
		log.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func UpdateEventsFromGencon(db *sql.DB, sourceFile string) {
	var events []*events.GenconEvent
	log.Printf("Loading events from %v", sourceFile)

	if strings.HasPrefix(sourceFile, "http") {
		events = parseHttp(sourceFile)
	} else if strings.HasSuffix(sourceFile, "xlsx") {
		events = parseSheet(sourceFile)
	} else {
		events = parseCsv(sourceFile)
	}

	writeEvents(db, events)
}
