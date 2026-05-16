package postgres

import (
	"database/sql"
	"flag"
	"os"
	"time"
)

var dbConnectString = flag.String("db", "", "postgres connect string")

var INDIANAPOLIS, _ = time.LoadLocation("America/Indiana/Indianapolis")

func GetConnStr() string {
	connStr := *dbConnectString
	if connStr == "" {
		connStr = os.Getenv("DATABASE_URL")
	}
	return connStr
}

func OpenDb() (*sql.DB, error) {
	return sql.Open("postgres", GetConnStr())
}
