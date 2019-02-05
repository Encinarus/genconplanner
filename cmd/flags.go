package main

import "flag"

var dbConnectString = flag.String("db", "", "postgres connect string")
var sourceFile = flag.String("eventFile", "", "file path or url to load from")
var searchQuery = flag.String("searchQuery", "True Dungeon -token", "a query to search the database on")
var eventId = flag.String("eventId", "TDA17117668", "a query to search the database on")
