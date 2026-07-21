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
	RemoveStarredEventGroup(email string, eventId string) (*postgres.UserStarredEvents, error)
	LoadStarredEvents(email string, year int) ([]*events.GenconEvent, error)
	LoadStarredEventGroups(email string, year int) ([]*postgres.EventGroup, error)
	LoadStarredEventClusters(email string, year int, starredEvents []*events.GenconEvent) ([]*postgres.CalendarEventCluster, error)
	ClearStarredEvents(email string, year int) error
	BulkStarEvents(email string, year int, eventIds []string, overwrite bool, asGroups bool, asPurchased bool) error
	LoadAgenda(email string, year int) ([]*postgres.AgendaEntry, error)
	GetWishlistConstraints(email string) ([]postgres.WishlistConstraint, error)
	UpdateWishlistConstraints(email string, constraints []postgres.WishlistConstraint) error
	GetWishlistCache(email string, year int) ([]postgres.WishlistCacheItem, bool, time.Time, error)
	SaveWishlistCache(email string, year int, items []postgres.WishlistCacheItem, updatedAt time.Time) error

	// Party related
	LoadParties(user *postgres.User) ([]*postgres.Party, error)
	LoadParty(id int64) (*postgres.Party, error)
	LoadPartyByCode(code string) (*postgres.Party, error)
	NewParty(name string, year int64, founderEmail string) (*postgres.Party, error)
	UpdatePartyLeader(id int64, newLeaderEmail string) error
	RenameParty(id int64, name string) error
	DeleteParty(id int64) error
	RemoveMember(partyId int64, email string) error
	JoinParty(partyId int64, email string) error
	LoadPartySharedInterests(partyId int64, year int) ([]*postgres.SharedInterestGroup, error)
	UpdatePartyMemberInfo(partyId int64, email string, displayName string, genconName string, genconId string, genconEmail string) error
	LoadPartyMemberPurchases(partyId int64, year int) (map[string]int, error)
	SyncPartyTickets(partyId int64, year int, authEmail string, tickets []postgres.TicketSyncInput) ([]*postgres.PartyTicket, error)
	LoadPartyTickets(partyId int64, year int) ([]*postgres.PartyTicket, error)
	AddPartyTicket(partyId int64, year int, eventId, purchaserEmail, genconRecipientName, holderEmail, ticketType string) (*postgres.PartyTicket, error)
	DeletePartyTicket(partyId int64, ticketId string) error
	InitiateTicketTransfer(partyId int64, ticketId, fromEmail, toEmail, transferType string) (*postgres.TicketTransfer, error)
	RespondTicketTransfer(partyId int64, transferId, action, callerEmail string) (*postgres.TicketTransfer, error)
	ToggleTicketReturn(partyId int64, ticketId string) (*postgres.PartyTicket, error)
	UpdateTicketPurchaser(partyId int64, ticketId string, newPurchaserEmail string) (*postgres.PartyTicket, error)

	// User related
	UpdateDisplayName(email string, name string) error
	UpdateUserGenconInfo(email string, displayName string, genconName string, genconId string, genconEmail string) error
	GetLastUpdate() (time.Time, error)

	// Admin related
	IsAdmin(email string) (bool, error)
	LoadAllOrgs() ([]*postgres.Organizer, error)
	MergeOrgs(orgs []int64) error
	LoadEventOrgMetadata() ([]postgres.EventOrgMetadata, error)
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

func (r *PostgresRepository) RemoveStarredEventGroup(email string, eventId string) (*postgres.UserStarredEvents, error) {
	return postgres.RemoveStarredEventGroup(r.DB, email, eventId)
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

func (r *PostgresRepository) BulkStarEvents(email string, year int, eventIds []string, overwrite bool, asGroups bool, asPurchased bool) error {
	return postgres.BulkStarEvents(r.DB, email, year, eventIds, overwrite, asGroups, asPurchased)
}

func (r *PostgresRepository) LoadAgenda(email string, year int) ([]*postgres.AgendaEntry, error) {
	return postgres.LoadAgenda(r.DB, email, year)
}

func (r *PostgresRepository) GetWishlistConstraints(email string) ([]postgres.WishlistConstraint, error) {
	return postgres.GetWishlistConstraints(r.DB, email)
}

func (r *PostgresRepository) UpdateWishlistConstraints(email string, constraints []postgres.WishlistConstraint) error {
	return postgres.UpdateWishlistConstraints(r.DB, email, constraints)
}

func (r *PostgresRepository) GetWishlistCache(email string, year int) ([]postgres.WishlistCacheItem, bool, time.Time, error) {
	return postgres.GetWishlistCache(r.DB, email, year)
}

func (r *PostgresRepository) SaveWishlistCache(email string, year int, items []postgres.WishlistCacheItem, updatedAt time.Time) error {
	return postgres.SaveWishlistCache(r.DB, email, year, items, updatedAt)
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

func (r *PostgresRepository) LoadPartyByCode(code string) (*postgres.Party, error) {
	return postgres.LoadPartyByCode(r.DB, code)
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

func (r *PostgresRepository) LoadPartySharedInterests(partyId int64, year int) ([]*postgres.SharedInterestGroup, error) {
	return postgres.LoadPartySharedInterests(r.DB, partyId, year)
}

func (r *PostgresRepository) UpdateDisplayName(email string, name string) error {
	return postgres.UpdateDisplayName(r.DB, email, name)
}

func (r *PostgresRepository) UpdateUserGenconInfo(email string, displayName string, genconName string, genconId string, genconEmail string) error {
	return postgres.UpdateUserGenconInfo(r.DB, email, displayName, genconName, genconId, genconEmail)
}

func (r *PostgresRepository) UpdatePartyMemberInfo(partyId int64, email string, displayName string, genconName string, genconId string, genconEmail string) error {
	return postgres.UpdatePartyMemberInfo(r.DB, partyId, email, displayName, genconName, genconId, genconEmail)
}

func (r *PostgresRepository) LoadPartyMemberPurchases(partyId int64, year int) (map[string]int, error) {
	return postgres.LoadPartyMemberPurchases(r.DB, partyId, year)
}

func (r *PostgresRepository) GetLastUpdate() (time.Time, error) {
	return postgres.GetLastUpdate(r.DB)
}

func (r *PostgresRepository) SyncPartyTickets(partyId int64, year int, authEmail string, tickets []postgres.TicketSyncInput) ([]*postgres.PartyTicket, error) {
	return postgres.SyncPartyTickets(r.DB, partyId, year, authEmail, tickets)
}

func (r *PostgresRepository) LoadPartyTickets(partyId int64, year int) ([]*postgres.PartyTicket, error) {
	return postgres.LoadPartyTickets(r.DB, partyId, year)
}

func (r *PostgresRepository) AddPartyTicket(partyId int64, year int, eventId, purchaserEmail, genconRecipientName, holderEmail, ticketType string) (*postgres.PartyTicket, error) {
	return postgres.AddPartyTicket(r.DB, partyId, year, eventId, purchaserEmail, genconRecipientName, holderEmail, ticketType)
}

func (r *PostgresRepository) DeletePartyTicket(partyId int64, ticketId string) error {
	return postgres.DeletePartyTicket(r.DB, partyId, ticketId)
}

func (r *PostgresRepository) InitiateTicketTransfer(partyId int64, ticketId, fromEmail, toEmail, transferType string) (*postgres.TicketTransfer, error) {
	return postgres.InitiateTicketTransfer(r.DB, partyId, ticketId, fromEmail, toEmail, transferType)
}

func (r *PostgresRepository) RespondTicketTransfer(partyId int64, transferId, action, callerEmail string) (*postgres.TicketTransfer, error) {
	return postgres.RespondTicketTransfer(r.DB, partyId, transferId, action, callerEmail)
}

func (r *PostgresRepository) ToggleTicketReturn(partyId int64, ticketId string) (*postgres.PartyTicket, error) {
	return postgres.ToggleTicketReturn(r.DB, partyId, ticketId)
}

func (r *PostgresRepository) UpdateTicketPurchaser(partyId int64, ticketId string, newPurchaserEmail string) (*postgres.PartyTicket, error) {
	return postgres.UpdateTicketPurchaser(r.DB, partyId, ticketId, newPurchaserEmail)
}

func (r *PostgresRepository) IsAdmin(email string) (bool, error) {
	return postgres.IsAdmin(r.DB, email)
}

func (r *PostgresRepository) LoadAllOrgs() ([]*postgres.Organizer, error) {
	return postgres.LoadAllOrgs(r.DB)
}

func (r *PostgresRepository) MergeOrgs(orgs []int64) error {
	return postgres.MergeOrgs(r.DB, orgs)
}

func (r *PostgresRepository) LoadEventOrgMetadata() ([]postgres.EventOrgMetadata, error) {
	return postgres.LoadEventOrgMetadata(r.DB)
}
