package main

import (
	"database/sql"
	"flag"
	"github.com/Encinarus/genconplanner/events"
	"github.com/Encinarus/genconplanner/postgres"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var dbConnectString = flag.String("db", "", "postgres connect string")
var sourceFile = flag.String("eventFile", "", "file path or url to load from")

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

	db, err := sql.Open("postgres", *dbConnectString)
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
		log.Fatalf("Downloading isn't implemented yet, use a local file")
	} else if strings.HasSuffix(*sourceFile, "xlsx") {
		events = parseSheet()
	} else {
		events = parseCsv()
	}

	writeEvents(db, events)
}
