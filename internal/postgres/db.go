package postgres

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"
)

var dbConnectString = flag.String("db", "", "postgres connect string")

var INDIANAPOLIS, _ = time.LoadLocation("America/Indiana/Indianapolis")

func OpenDb() (*sql.DB, error) {
	connStr := *dbConnectString
	if connStr == "" {
		connStr = os.Getenv("DATABASE_URL")
	}
	fmt.Println("dbString", connStr)
	return sql.Open("postgres", connStr)
}
