package api

import (
	"database/sql"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
)

type EventRepository interface {
	LoadCategorySummary(year int) ([]*postgres.CategorySummary, error)
	LoadEventGroupsForCategory(category string, year int) ([]*postgres.EventGroup, error)
	SearchEvents(q postgres.SearchQuery) ([]*postgres.EventGroup, error)
	LoadSimilarEvents(eventId string, userEmail string) ([]*events.GenconEvent, error)
	LoadOrCreateUser(email string) (*postgres.User, error)
	GetStarredIds(email string, year int) (*postgres.UserStarredEvents, error)
	GetAllStarredIds(email string) (*postgres.UserStarredEvents, error)
	UpdateStarredEvent(email string, eventId string, tier string, starGroup bool, add bool) (*postgres.UserStarredEvents, error)
	UpdateStarredEventMinimal(email string, eventId string, tier string, starGroup bool, add bool) (*postgres.UserStarredEvents, error)
	LoadStarredEvents(email string, year int) ([]*events.GenconEvent, error)
	LoadStarredEventGroups(email string, year int) ([]*postgres.EventGroup, error)
	LoadStarredEventClusters(email string, year int, starredEvents []*events.GenconEvent) ([]*postgres.CalendarEventCluster, error)
	ClearStarredEvents(email string, year int) error
	BulkStarEvents(email string, year int, eventIds []string, overwrite bool, asGroups bool) error
	LoadAgenda(email string, year int) ([]*postgres.AgendaEntry, error)

	// Party related
	LoadParties(user *postgres.User) ([]*postgres.Party, error)
	LoadParty(id int64) (*postgres.Party, error)
	NewParty(name string, year int64, founderEmail string) (*postgres.Party, error)
	UpdatePartyLeader(id int64, newLeaderEmail string) error
	RenameParty(id int64, name string) error
	DeleteParty(id int64) error
	RemoveMember(partyId int64, email string) error
	JoinParty(partyId int64, email string) error

	// User related
	UpdateDisplayName(email string, name string) error
	GetLastUpdate() (time.Time, error)
}

type PostgresRepository struct {
	DB *sql.DB
}

func (r *PostgresRepository) LoadCategorySummary(year int) ([]*postgres.CategorySummary, error) {
	return postgres.LoadCategorySummary(r.DB, year)
}

func (r *PostgresRepository) LoadEventGroupsForCategory(category string, year int) ([]*postgres.EventGroup, error) {
	return postgres.LoadEventGroupsForCategory(r.DB, category, year)
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
func (r *PostgresRepository) UpdateStarredEvent(email string, eventId string, tier string, starGroup bool, add bool) (*postgres.UserStarredEvents, error) {
	return postgres.UpdateStarredEvent(r.DB, email, eventId, tier, starGroup, add)
}

func (r *PostgresRepository) UpdateStarredEventMinimal(email string, eventId string, tier string, starGroup bool, add bool) (*postgres.UserStarredEvents, error) {
	return postgres.UpdateStarredEventMinimal(r.DB, email, eventId, tier, starGroup, add)
}

func (r *PostgresRepository) LoadStarredEvents(email string, year int) ([]*events.GenconEvent, error) {
	return postgres.LoadStarredEvents(r.DB, email, year)
}

func (r *PostgresRepository) LoadStarredEventGroups(email string, year int) ([]*postgres.EventGroup, error) {
	return postgres.LoadStarredEventGroups(r.DB, email, year)
}

func (r *PostgresRepository) LoadStarredEventClusters(email string, year int, starredEvents []*events.GenconEvent) ([]*postgres.CalendarEventCluster, error) {
	return postgres.LoadStarredEventClusters(r.DB, email, year, starredEvents)
}

func (r *PostgresRepository) ClearStarredEvents(email string, year int) error {
	return postgres.ClearStarredEvents(r.DB, email, year)
}

func (r *PostgresRepository) BulkStarEvents(email string, year int, eventIds []string, overwrite bool, asGroups bool) error {
	return postgres.BulkStarEvents(r.DB, email, year, eventIds, overwrite, asGroups)
}

func (r *PostgresRepository) LoadAgenda(email string, year int) ([]*postgres.AgendaEntry, error) {
	return postgres.LoadAgenda(r.DB, email, year)
}

func (r *PostgresRepository) LoadParties(user *postgres.User) ([]*postgres.Party, error) {
	return postgres.LoadParties(r.DB, user)
}

func (r *PostgresRepository) NewParty(name string, year int64, founderEmail string) (*postgres.Party, error) {
	return postgres.NewParty(r.DB, name, year, founderEmail)
}

func (r *PostgresRepository) LoadParty(id int64) (*postgres.Party, error) {
	return postgres.LoadParty(r.DB, id)
}

func (r *PostgresRepository) UpdatePartyLeader(id int64, newLeaderEmail string) error {
	return postgres.UpdatePartyLeader(r.DB, id, newLeaderEmail)
}

func (r *PostgresRepository) RenameParty(id int64, name string) error {
	return postgres.RenameParty(r.DB, id, name)
}

func (r *PostgresRepository) DeleteParty(id int64) error {
	return postgres.DeleteParty(r.DB, id)
}

func (r *PostgresRepository) RemoveMember(partyId int64, email string) error {
	return postgres.RemoveMember(r.DB, partyId, email)
}

func (r *PostgresRepository) JoinParty(partyId int64, email string) error {
	return postgres.JoinParty(r.DB, partyId, email)
}

func (r *PostgresRepository) UpdateDisplayName(email string, name string) error {
	return postgres.UpdateDisplayName(r.DB, email, name)
}

func (r *PostgresRepository) GetLastUpdate() (time.Time, error) {
	return postgres.GetLastUpdate(r.DB)
}

