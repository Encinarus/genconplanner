package main

import (
	"database/sql"
	"flag"
	"github.com/Encinarus/genconplanner/postgres"
	"log"
)


var dbConnectString = flag.String("db", "", "postgres connect string")
var searchQuery = flag.String("searchQuery", "True Dungeon -token", "a query to search the database on")

func main() {
	flag.Parse()

	db, err := sql.Open("postgres", *dbConnectString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	foundEvents := postgres.FindEvents(db, *searchQuery)
	log.Printf("%v events found", len(foundEvents))

	for _, event := range foundEvents {
		log.Println(event.EventId)
	}
}