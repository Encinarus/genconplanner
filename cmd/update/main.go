package main

import (
	"database/sql"
	"flag"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var sourceFile = flag.String("eventFile", "https://www.gencon.com/downloads/events.xlsx", "file path or url to load from")

func parseHttp() []*events.GenconEvent {
	resp, err := http.Get(*sourceFile)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	spreadsheetBytes, err := ioutil.ReadAll(resp.Body)
	return events.ParseGenconSheet(spreadsheetBytes)
}

func parseSheet() []*events.GenconEvent {
	fileReader, err := os.Open(*sourceFile)

	if err != nil {
		panic(err)
	}
	defer fileReader.Close()
	fileBytes, err := ioutil.ReadAll(fileReader)

	return events.ParseGenconSheet(fileBytes)
}

func parseCsv() []*events.GenconEvent {
	fileReader, err := os.Open(*sourceFile)

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

func main() {
	flag.Parse()

	db, err := postgres.OpenDb()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var events []*events.GenconEvent
	log.Printf("Loading events from %v", *sourceFile)

	if len(*sourceFile) == 0 {
		log.Fatalf("You must specify a source file")
	}
	if strings.HasPrefix(*sourceFile, "http") {
		events = parseHttp()
		// log.Fatalf("Downloading isn't implemented yet, use a local file")

	} else if strings.HasSuffix(*sourceFile, "xlsx") {
		events = parseSheet()
	} else {
		events = parseCsv()
	}

	writeEvents(db, events)
}
