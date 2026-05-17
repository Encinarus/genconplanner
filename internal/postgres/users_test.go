package postgres

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockQueryable struct {
	db *sql.DB
}

func (m *mockQueryable) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return m.db.Query(query, args...)
}

func (m *mockQueryable) Exec(query string, args ...interface{}) (sql.Result, error) {
	return m.db.Exec(query, args...)
}

func TestNormalizeUserStarredEvents_Property1(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	q := &mockQueryable{db: db}
	email := "test@example.com"
	year := 2026

	// Mock the SELECT query finding active groups and their starred instances
	rows := sqlmock.NewRows([]string{"year", "short_category", "title", "short_description", "all_event_ids", "starred_levels", "starred_tiers"}).
		AddRow(2026, "BGM", "Catan", "Trading game",
			"{BGM26ND100001,BGM26ND100002}",
			"{group,group}",
			"{very_interested,must_have}")

	mock.ExpectQuery("SELECT e.year, e.short_category, e.title").
		WithArgs(email, year).
		WillReturnRows(rows)

	// Since BGM26ND100002 has 'must_have' (priority 4) vs BGM26ND100001 'very_interested' (priority 3),
	// BGM26ND100001 should be demoted to 'event'
	mock.ExpectExec("INSERT INTO starred_events").
		WithArgs(email, "BGM26ND100001").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = NormalizeUserStarredEvents(q, email, year)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestNormalizeUserStarredEvents_Property2(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	q := &mockQueryable{db: db}
	email := "test@example.com"
	year := 2026

	// Mock the SELECT query finding an active group where NO instance holds level = 'group'.
	// BGM26ND100001 is unstarred (NULL), BGM26ND100002 is level='event', tier='must_have'.
	rows := sqlmock.NewRows([]string{"year", "short_category", "title", "short_description", "all_event_ids", "starred_levels", "starred_tiers"}).
		AddRow(2026, "BGM", "Catan", "Trading game",
			"{BGM26ND100001,BGM26ND100002}",
			"{NULL,event}",
			"{NULL,must_have}")

	mock.ExpectQuery("SELECT e.year, e.short_category, e.title").
		WithArgs(email, year).
		WillReturnRows(rows)

	// Step 2: lowest sorting event ID (BGM26ND100001) gets level='group', tier='must_have' (max tier)
	mock.ExpectExec("INSERT INTO starred_events").
		WithArgs(email, "BGM26ND100001", "must_have").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = NormalizeUserStarredEvents(q, email, year)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestFetchStarredInternal_Property3(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	q := &mockQueryable{db: db}
	email := "test@example.com"
	year := 2026

	// First, NormalizeUserStarredEvents runs
	normRows := sqlmock.NewRows([]string{"year", "short_category", "title", "short_description", "all_event_ids", "starred_levels", "starred_tiers"}).
		AddRow(2026, "BGM", "Catan", "Trading game",
			"{BGM26ND100001,BGM26ND100002}",
			"{group,event}",
			"{very_interested,must_have}")

	mock.ExpectQuery("SELECT e.year, e.short_category, e.title").
		WithArgs(email, year).
		WillReturnRows(normRows)

	// Next, fetchStarredInternal executes the JOIN query
	fetchRows := sqlmock.NewRows([]string{"event_id", "level", "tier", "group_tier", "is_override"}).
		AddRow("BGM26ND100001", "group", "very_interested", "very_interested", false).
		AddRow("BGM26ND100002", "event", "must_have", "very_interested", true)

	mock.ExpectQuery("SELECT e2.event_id").
		WithArgs(email, year).
		WillReturnRows(fetchRows)

	res, err := fetchStarredInternal(q, email, year)
	if err != nil {
		t.Fatalf("error was not expected: %s", err)
	}

	if len(res.StarredEvents) != 2 {
		t.Fatalf("expected 2 starred events, got %d", len(res.StarredEvents))
	}

	se1 := res.StarredEvents[0]
	if se1.EventId != "BGM26ND100001" || se1.GroupTier != "very_interested" || se1.IsOverride != false {
		t.Errorf("unexpected values for se1: %+v", se1)
	}

	se2 := res.StarredEvents[1]
	if se2.EventId != "BGM26ND100002" || se2.GroupTier != "very_interested" || se2.IsOverride != true {
		t.Errorf("unexpected values for se2: %+v", se2)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
