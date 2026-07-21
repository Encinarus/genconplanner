package api

import (
	"context"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

// StubRepository implements EventRepository for testing.
type StubRepository struct {
	LoadCategorySummaryFn        func(int) ([]*postgres.CategorySummary, error)
	LoadEventGroupsForCategoryFn func(string, int) ([]*postgres.EventGroup, error)
	SearchEventsFn               func(postgres.SearchQuery) ([]*postgres.EventGroup, error)
	LoadSimilarEventsFn          func(string, string) ([]*events.GenconEvent, error)
	LoadOrCreateUserFn           func(string) (*postgres.User, error)
	GetStarredIdsFn              func(string, int) (*postgres.UserStarredEvents, error)
	GetAllStarredIdsFn           func(string) (*postgres.UserStarredEvents, error)
	UpdateStarredEventFn         func(string, string, string, bool, bool) (*postgres.UserStarredEvents, error)
	UpdateStarredEventMinimalFn  func(string, string, string, bool, bool) (*postgres.UserStarredEvents, error)
	RemoveStarredEventGroupFn    func(string, string) (*postgres.UserStarredEvents, error)
	LoadStarredEventsFn          func(string, int) ([]*events.GenconEvent, error)
	LoadStarredEventGroupsFn     func(string, int) ([]*postgres.EventGroup, error)
	LoadStarredEventClustersFn   func(string, int, []*events.GenconEvent) ([]*postgres.CalendarEventCluster, error)
	ClearStarredEventsFn         func(string, int) error
	BulkStarEventsFn             func(string, int, []string, bool, bool, bool) error
	LoadAgendaFn                 func(string, int) ([]*postgres.AgendaEntry, error)
	GetWishlistConstraintsFn     func(string) ([]postgres.WishlistConstraint, error)
	UpdateWishlistConstraintsFn  func(string, []postgres.WishlistConstraint) error
	LoadPartiesFn                func(*postgres.User) ([]*postgres.Party, error)
	LoadPartyFn                  func(int64) (*postgres.Party, error)
	LoadPartyByCodeFn            func(string) (*postgres.Party, error)
	NewPartyFn                   func(string, int64, string) (*postgres.Party, error)
	UpdatePartyLeaderFn          func(int64, string) error
	RenamePartyFn                func(int64, string) error
	DeletePartyFn                func(int64) error
	RemoveMemberFn               func(int64, string) error
	JoinPartyFn                  func(int64, string) error
	LoadPartySharedInterestsFn   func(int64, int) ([]*postgres.SharedInterestGroup, error)
	UpdatePartyMemberInfoFn      func(int64, string, string, string, string, string) error
	LoadPartyMemberPurchasesFn   func(int64, int) (map[string]int, error)
	UpdateDisplayNameFn          func(string, string) error
	UpdateUserGenconInfoFn       func(string, string, string, string, string) error
	GetLastUpdateFn              func() (time.Time, error)
	GetWishlistCacheFn           func(string, int) ([]postgres.WishlistCacheItem, bool, time.Time, error)
	SaveWishlistCacheFn          func(string, int, []postgres.WishlistCacheItem, time.Time) error
	SyncPartyTicketsFn           func(int64, int, string, []postgres.TicketSyncInput) ([]*postgres.PartyTicket, error)
	LoadPartyTicketsFn           func(int64, int) ([]*postgres.PartyTicket, error)
	AddPartyTicketFn             func(int64, int, string, string, string, string, string) (*postgres.PartyTicket, error)
	DeletePartyTicketFn          func(int64, string) error
	InitiateTicketTransferFn     func(int64, string, string, string, string) (*postgres.TicketTransfer, error)
	RespondTicketTransferFn      func(int64, string, string, string) (*postgres.TicketTransfer, error)
	ToggleTicketReturnFn         func(int64, string) (*postgres.PartyTicket, error)
	IsAdminFn                    func(string) (bool, error)
	LoadAllOrgsFn                func() ([]*postgres.Organizer, error)
	MergeOrgsFn                  func([]int64) error
	LoadEventOrgMetadataFn       func() ([]postgres.EventOrgMetadata, error)
}

func (s *StubRepository) LoadCategorySummary(year int) ([]*postgres.CategorySummary, error) {
	return s.LoadCategorySummaryFn(year)
}
func (s *StubRepository) LoadEventGroupsForCategory(category string, year int) ([]*postgres.EventGroup, error) {
	return s.LoadEventGroupsForCategoryFn(category, year)
}
func (s *StubRepository) SearchEvents(q postgres.SearchQuery) ([]*postgres.EventGroup, error) {
	return s.SearchEventsFn(q)
}
func (s *StubRepository) LoadSimilarEvents(eventId string, userEmail string) ([]*events.GenconEvent, error) {
	return s.LoadSimilarEventsFn(eventId, userEmail)
}
func (s *StubRepository) LoadOrCreateUser(email string) (*postgres.User, error) {
	return s.LoadOrCreateUserFn(email)
}
func (s *StubRepository) GetStarredIds(email string, year int) (*postgres.UserStarredEvents, error) {
	return s.GetStarredIdsFn(email, year)
}
func (s *StubRepository) GetAllStarredIds(email string) (*postgres.UserStarredEvents, error) {
	return s.GetAllStarredIdsFn(email)
}
func (s *StubRepository) UpdateStarredEvent(email string, eventId string, tier string, starGroup bool, add bool) (*postgres.UserStarredEvents, error) {
	return s.UpdateStarredEventFn(email, eventId, tier, starGroup, add)
}
func (s *StubRepository) UpdateStarredEventMinimal(email string, eventId string, tier string, starGroup bool, add bool) (*postgres.UserStarredEvents, error) {
	return s.UpdateStarredEventMinimalFn(email, eventId, tier, starGroup, add)
}
func (s *StubRepository) RemoveStarredEventGroup(email string, eventId string) (*postgres.UserStarredEvents, error) {
	if s.RemoveStarredEventGroupFn == nil {
		return nil, nil
	}
	return s.RemoveStarredEventGroupFn(email, eventId)
}
func (s *StubRepository) LoadStarredEvents(email string, year int) ([]*events.GenconEvent, error) {
	return s.LoadStarredEventsFn(email, year)
}
func (s *StubRepository) LoadStarredEventGroups(email string, year int) ([]*postgres.EventGroup, error) {
	return s.LoadStarredEventGroupsFn(email, year)
}
func (s *StubRepository) LoadStarredEventClusters(email string, year int, starredEvents []*events.GenconEvent) ([]*postgres.CalendarEventCluster, error) {
	return s.LoadStarredEventClustersFn(email, year, starredEvents)
}
func (s *StubRepository) ClearStarredEvents(email string, year int) error {
	return s.ClearStarredEventsFn(email, year)
}
func (s *StubRepository) BulkStarEvents(email string, year int, eventIds []string, overwrite bool, asGroups bool, asPurchased bool) error {
	return s.BulkStarEventsFn(email, year, eventIds, overwrite, asGroups, asPurchased)
}
func (s *StubRepository) LoadAgenda(email string, year int) ([]*postgres.AgendaEntry, error) {
	return s.LoadAgendaFn(email, year)
}
func (s *StubRepository) GetWishlistConstraints(email string) ([]postgres.WishlistConstraint, error) {
	if s.GetWishlistConstraintsFn == nil {
		return nil, nil
	}
	return s.GetWishlistConstraintsFn(email)
}
func (s *StubRepository) UpdateWishlistConstraints(email string, constraints []postgres.WishlistConstraint) error {
	if s.UpdateWishlistConstraintsFn == nil {
		return nil
	}
	return s.UpdateWishlistConstraintsFn(email, constraints)
}
func (s *StubRepository) LoadParties(user *postgres.User) ([]*postgres.Party, error) {
	return s.LoadPartiesFn(user)
}
func (s *StubRepository) LoadParty(id int64) (*postgres.Party, error) {
	if s.LoadPartyFn == nil {
		return nil, nil
	}
	return s.LoadPartyFn(id)
}
func (s *StubRepository) LoadPartyByCode(code string) (*postgres.Party, error) {
	if s.LoadPartyByCodeFn == nil {
		return nil, nil
	}
	return s.LoadPartyByCodeFn(code)
}
func (s *StubRepository) NewParty(name string, year int64, founderEmail string) (*postgres.Party, error) {
	return s.NewPartyFn(name, year, founderEmail)
}
func (s *StubRepository) UpdatePartyLeader(id int64, newLeaderEmail string) error {
	if s.UpdatePartyLeaderFn == nil {
		return nil
	}
	return s.UpdatePartyLeaderFn(id, newLeaderEmail)
}
func (s *StubRepository) RenameParty(id int64, name string) error {
	if s.RenamePartyFn == nil {
		return nil
	}
	return s.RenamePartyFn(id, name)
}
func (s *StubRepository) DeleteParty(id int64) error {
	if s.DeletePartyFn == nil {
		return nil
	}
	return s.DeletePartyFn(id)
}
func (s *StubRepository) RemoveMember(partyId int64, email string) error {
	if s.RemoveMemberFn == nil {
		return nil
	}
	return s.RemoveMemberFn(partyId, email)
}
func (s *StubRepository) JoinParty(partyId int64, email string) error {
	if s.JoinPartyFn == nil {
		return nil
	}
	return s.JoinPartyFn(partyId, email)
}

func (s *StubRepository) LoadPartySharedInterests(partyId int64, year int) ([]*postgres.SharedInterestGroup, error) {
	if s.LoadPartySharedInterestsFn == nil {
		return nil, nil
	}
	return s.LoadPartySharedInterestsFn(partyId, year)
}
func (s *StubRepository) UpdatePartyMemberInfo(partyId int64, email string, displayName string, genconName string, genconId string, genconEmail string) error {
	if s.UpdatePartyMemberInfoFn == nil {
		return nil
	}
	return s.UpdatePartyMemberInfoFn(partyId, email, displayName, genconName, genconId, genconEmail)
}
func (s *StubRepository) LoadPartyMemberPurchases(partyId int64, year int) (map[string]int, error) {
	if s.LoadPartyMemberPurchasesFn == nil {
		return make(map[string]int), nil
	}
	return s.LoadPartyMemberPurchasesFn(partyId, year)
}
func (s *StubRepository) UpdateDisplayName(email string, name string) error {
	return s.UpdateDisplayNameFn(email, name)
}
func (s *StubRepository) UpdateUserGenconInfo(email string, displayName string, genconName string, genconId string, genconEmail string) error {
	if s.UpdateUserGenconInfoFn == nil {
		return nil
	}
	return s.UpdateUserGenconInfoFn(email, displayName, genconName, genconId, genconEmail)
}
func (s *StubRepository) GetLastUpdate() (time.Time, error) {
	if s.GetLastUpdateFn != nil {
		return s.GetLastUpdateFn()
	}
	return time.Time{}, nil
}
func (s *StubRepository) GetWishlistCache(email string, year int) ([]postgres.WishlistCacheItem, bool, time.Time, error) {
	if s.GetWishlistCacheFn == nil {
		return nil, true, time.Time{}, nil
	}
	return s.GetWishlistCacheFn(email, year)
}

func (s *StubRepository) SaveWishlistCache(email string, year int, items []postgres.WishlistCacheItem, updatedAt time.Time) error {
	if s.SaveWishlistCacheFn == nil {
		return nil
	}
	return s.SaveWishlistCacheFn(email, year, items, updatedAt)
}

func (s *StubRepository) SyncPartyTickets(partyId int64, year int, authEmail string, tickets []postgres.TicketSyncInput) ([]*postgres.PartyTicket, error) {
	if s.SyncPartyTicketsFn == nil {
		return nil, nil
	}
	return s.SyncPartyTicketsFn(partyId, year, authEmail, tickets)
}

func (s *StubRepository) LoadPartyTickets(partyId int64, year int) ([]*postgres.PartyTicket, error) {
	if s.LoadPartyTicketsFn == nil {
		return nil, nil
	}
	return s.LoadPartyTicketsFn(partyId, year)
}

func (s *StubRepository) AddPartyTicket(partyId int64, year int, eventId, purchaserEmail, genconRecipientName, holderEmail, ticketType string) (*postgres.PartyTicket, error) {
	if s.AddPartyTicketFn == nil {
		return nil, nil
	}
	return s.AddPartyTicketFn(partyId, year, eventId, purchaserEmail, genconRecipientName, holderEmail, ticketType)
}

func (s *StubRepository) DeletePartyTicket(partyId int64, ticketId string) error {
	if s.DeletePartyTicketFn == nil {
		return nil
	}
	return s.DeletePartyTicketFn(partyId, ticketId)
}

func (s *StubRepository) InitiateTicketTransfer(partyId int64, ticketId, fromEmail, toEmail, transferType string) (*postgres.TicketTransfer, error) {
	if s.InitiateTicketTransferFn == nil {
		return nil, nil
	}
	return s.InitiateTicketTransferFn(partyId, ticketId, fromEmail, toEmail, transferType)
}

func (s *StubRepository) RespondTicketTransfer(partyId int64, transferId, action, callerEmail string) (*postgres.TicketTransfer, error) {
	if s.RespondTicketTransferFn == nil {
		return nil, nil
	}
	return s.RespondTicketTransferFn(partyId, transferId, action, callerEmail)
}

func (s *StubRepository) ToggleTicketReturn(partyId int64, ticketId string) (*postgres.PartyTicket, error) {
	if s.ToggleTicketReturnFn == nil {
		return nil, nil
	}
	return s.ToggleTicketReturnFn(partyId, ticketId)
}

func (s *StubRepository) IsAdmin(email string) (bool, error) {
	if s.IsAdminFn == nil {
		return false, nil
	}
	return s.IsAdminFn(email)
}

func (s *StubRepository) LoadAllOrgs() ([]*postgres.Organizer, error) {
	if s.LoadAllOrgsFn == nil {
		return nil, nil
	}
	return s.LoadAllOrgsFn()
}

func (s *StubRepository) MergeOrgs(orgs []int64) error {
	if s.MergeOrgsFn == nil {
		return nil
	}
	return s.MergeOrgsFn(orgs)
}

func (s *StubRepository) LoadEventOrgMetadata() ([]postgres.EventOrgMetadata, error) {
	if s.LoadEventOrgMetadataFn == nil {
		return nil, nil
	}
	return s.LoadEventOrgMetadataFn()
}

// StubAuthService implements AuthService for testing.
type StubAuthService struct {
	VerifyIDTokenFn func(context.Context, string) (string, error)
}

func (s *StubAuthService) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	return s.VerifyIDTokenFn(ctx, idToken)
}

// StubGameService implements GameService for testing.
type StubGameService struct {
	FindGameFn func(string) *postgres.Game
}

func (s *StubGameService) FindGame(name string) *postgres.Game {
	return s.FindGameFn(name)
}

// setupTestServer returns a configured Gin engine and stubs for testing.
func setupTestServer() (*Server, *StubRepository, *StubAuthService, *StubGameService, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	repo := &StubRepository{}
	auth := &StubAuthService{}
	games := &StubGameService{}

	server := NewServer(repo, auth, games)
	router := gin.New()

	return server, repo, auth, games, router
}
