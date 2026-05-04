package api

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresRepository_LoadCategorySummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := &PostgresRepository{DB: db}
	year := 2024

	// Expect the query that is hardcoded in postgres.LoadCategorySummary
	mock.ExpectQuery("SELECT event_type, COUNT\\(1\\)").
		WithArgs(year).
		WillReturnRows(sqlmock.NewRows([]string{"event_type", "count"}).
			AddRow("BGM Board Games", 42).
			AddRow("RPG Role Playing Games", 10))

	summaries, err := repo.LoadCategorySummary(year)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if len(summaries) != 2 {
		t.Errorf("expected 2 summaries, got %d", len(summaries))
	}

	if summaries[0].Name != "BGM Board Games" || summaries[0].Count != 42 {
		t.Errorf("unexpected summary data at index 0")
	}

	if summaries[1].Code != "RPG" {
		t.Errorf("expected code RPG, got %s", summaries[1].Code)
	}

	// Make sure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
