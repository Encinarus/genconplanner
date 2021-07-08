package main

import (
	"flag"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"log"
)

var sourceFile = flag.String("eventFile", "https://www.gencon.com/downloads/events.xlsx", "file path or url to load from")

func main() {
	flag.Parse()

	db, err := postgres.OpenDb()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if len(*sourceFile) == 0 {
		log.Fatalf("You must specify a source file")
	}

	background.UpdateEventsFromGencon(db, *sourceFile)
}
