package api

import (
	"database/sql"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
)

type EventRepository interface {
	LoadCategorySummary(year int) ([]*postgres.CategorySummary, error)
	SearchEvents(q postgres.SearchQuery) ([]*postgres.EventGroup, error)
	LoadSimilarEvents(eventId string, userEmail string) ([]*events.GenconEvent, error)
	LoadOrCreateUser(email string) (*postgres.User, error)
	GetStarredIds(email string, year int) (*postgres.UserStarredEvents, error)
	GetAllStarredIds(email string) (*postgres.UserStarredEvents, error)
	UpdateStarredEvent(email string, eventId string, starGroup bool, add bool) (*postgres.UserStarredEvents, error)
	LoadStarredEvents(email string, year int) ([]*events.GenconEvent, error)
	LoadStarredEventGroups(email string, year int) ([]*postgres.EventGroup, error)
}

type PostgresRepository struct {
	DB *sql.DB
}

func (r *PostgresRepository) LoadCategorySummary(year int) ([]*postgres.CategorySummary, error) {
	return postgres.LoadCategorySummary(r.DB, year)
}

func (r *PostgresRepository) SearchEvents(q postgres.SearchQuery) ([]*postgres.EventGroup, error) {
	return postgres.SearchEvents(r.DB, q)
}

func (r *PostgresRepository) LoadSimilarEvents(eventId string, userEmail string) ([]*events.GenconEvent, error) {
	return postgres.LoadSimilarEvents(r.DB, eventId, userEmail)
}

func (r *PostgresRepository) LoadOrCreateUser(email string) (*postgres.User, error) {
	return postgres.LoadOrCreateUser(r.DB, email)
}

func (r *PostgresRepository) GetStarredIds(email string, year int) (*postgres.UserStarredEvents, error) {
	return postgres.GetStarredIds(r.DB, email, year)
}

func (r *PostgresRepository) GetAllStarredIds(email string) (*postgres.UserStarredEvents, error) {
	return postgres.GetAllStarredIds(r.DB, email)
}
func (r *PostgresRepository) UpdateStarredEvent(email string, eventId string, starGroup bool, add bool) (*postgres.UserStarredEvents, error) {
	return postgres.UpdateStarredEvent(r.DB, email, eventId, starGroup, add)
}

func (r *PostgresRepository) LoadStarredEvents(email string, year int) ([]*events.GenconEvent, error) {
	return postgres.LoadStarredEvents(r.DB, email, year)
}

func (r *PostgresRepository) LoadStarredEventGroups(email string, year int) ([]*postgres.EventGroup, error) {
	return postgres.LoadStarredEventGroups(r.DB, email, year)
}
