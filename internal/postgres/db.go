package postgres

import (
	"database/sql"
	"flag"
	"os"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
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
	return otelsql.Open("postgres", GetConnStr(),
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
	)
}
