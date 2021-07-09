package postgres

import (
	"database/sql"
	"flag"
	"fmt"
	"time"
)

var dbConnectString = flag.String("db", "", "postgres connect string")

var INDIANAPOLIS, _ = time.LoadLocation("America/Indiana/Indianapolis")

func OpenDb() (*sql.DB, error) {
	fmt.Println("dbString", *dbConnectString)
	return sql.Open("postgres", *dbConnectString)
}
