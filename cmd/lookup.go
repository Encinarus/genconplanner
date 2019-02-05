package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"github.com/Encinarus/genconplanner/events"
	"github.com/Encinarus/genconplanner/postgres"
	"log"
)


var dbConnectString = flag.String("db", "", "postgres connect string")
var eventId = flag.String("eventId", "TDA17117668", "a query to search the database on")

type LookupResult struct {
	MainEvent *events.GenconEvent
	SimilarEvents []*events.SlimEvent
}

func main() {
	flag.Parse()

	db, err := sql.Open("postgres", *dbConnectString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	foundEvents, err := postgres.LoadSimilarEvents(db, *eventId)
	if err != nil {
		log.Fatalf("Unable to load events, err %v", err)
	}
	log.Printf("Found %v events similar to %s", len(foundEvents), *eventId)

	var result LookupResult

	for _, event := range foundEvents {
		if event.EventId == *eventId {
			result.MainEvent = event
		} else {
			result.SimilarEvents = append(result.SimilarEvents, event.SlimEvent())
		}
	}

	jsonEvent, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("Unable to marshal event to json %v", err)
	}
	log.Printf("%+v", string(jsonEvent))
}