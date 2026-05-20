//go:build integration

package postgres_test

import (
	"testing"

	"github.com/Encinarus/genconplanner/internal/postgres"
)

func TestSyncPartyTickets_Integration(t *testing.T) {
	repo := setupSeededDB(t)
	db := repo.DB

	// Sync 2 tickets for leader@example.com
	// Ticket 1: Recipient is member1 (GC1002)
	// Ticket 2: Recipient is solo (GC1004) - not in party 101 yet!
	ticketsInput := []postgres.TicketSyncInput{
		{
			EventId:           "RPG26ND200001",
			GenconTicketId:    "TXN100-1",
			PurchaserGenconId: "GC1001",
			PurchaserName:     "Party Leader",
			RecipientGenconId: "GC1002",
			RecipientName:     "Active Member",
			TicketType:        "physical",
		},
		{
			EventId:           "RPG26ND200002",
			GenconTicketId:    "TXN100-2",
			PurchaserGenconId: "GC1001",
			PurchaserName:     "Party Leader",
			RecipientGenconId: "GC1004",
			RecipientName:     "Solo User",
			TicketType:        "physical",
		},
	}

	tickets, err := postgres.SyncPartyTickets(db, 101, 2026, "leader@example.com", ticketsInput)
	if err != nil {
		t.Fatalf("SyncPartyTickets failed: %v", err)
	}

	if len(tickets) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(tickets))
	}

	var t1, t2 *postgres.PartyTicket
	for _, pt := range tickets {
		if pt.GenconTicketId == "TXN100-1" {
			t1 = pt
		} else if pt.GenconTicketId == "TXN100-2" {
			t2 = pt
		}
	}

	if t1 == nil || t2 == nil {
		t.Fatalf("missing synced tickets")
	}

	// Ticket 1 should be held by member1
	if t1.HolderEmail != "member1@example.com" {
		t.Errorf("expected t1 holder member1@example.com, got %s", t1.HolderEmail)
	}

	// Ticket 2 should default to leader because solo is not in party 101
	if t2.HolderEmail != "leader@example.com" {
		t.Errorf("expected t2 holder leader@example.com, got %s", t2.HolderEmail)
	}

	// Verify member1 starred_events has tier = 'purchased' for RPG26ND200001
	starred, errSt := postgres.GetStarredIds(db, "member1@example.com", 2026)
	if errSt != nil {
		t.Fatalf("GetStarredIds failed: %v", errSt)
	}
	if tier := starred.GetTier("RPG26ND200001"); tier != "purchased" {
		t.Errorf("expected member1 tier 'purchased', got '%s'", tier)
	}
}

func TestInitiateTicketTransfer_Integration(t *testing.T) {
	repo := setupSeededDB(t)
	db := repo.DB

	// First add a ticket manually for leader
	pt, errAdd := postgres.AddPartyTicket(db, 101, 2026, "BGM26ND100001", "leader@example.com", "Party Leader", "leader@example.com", "physical")
	if errAdd != nil {
		t.Fatalf("AddPartyTicket failed: %v", errAdd)
	}

	// Initiate name_only transfer to member2
	tt, errTrans := postgres.InitiateTicketTransfer(db, 101, pt.TicketId, "leader@example.com", "member2@example.com", "name_only")
	if errTrans != nil {
		t.Fatalf("InitiateTicketTransfer failed: %v", errTrans)
	}

	if tt.Status != "completed" {
		t.Errorf("expected transfer status 'completed', got %s", tt.Status)
	}

	// Verify ticket holder is now member2
	tickets, _ := postgres.LoadPartyTickets(db, 101, 2026)
	var updatedTicket *postgres.PartyTicket
	for _, t := range tickets {
		if t.TicketId == pt.TicketId {
			updatedTicket = t
		}
	}
	if updatedTicket.HolderEmail != "member2@example.com" {
		t.Errorf("expected holder member2@example.com, got %s", updatedTicket.HolderEmail)
	}

	// Verify leader's starred event reverted to must_have
	starredLeader, _ := postgres.GetStarredIds(db, "leader@example.com", 2026)
	if tier := starredLeader.GetTier("BGM26ND100001"); tier != "must_have" {
		t.Errorf("expected leader tier 'must_have', got '%s'", tier)
	}
}

func TestMatchPendingPartyTickets_Integration(t *testing.T) {
	repo := setupSeededDB(t)
	db := repo.DB

	// 1. Sync ticket for solo (GC1004) while solo is not in party 101
	ticketsInput := []postgres.TicketSyncInput{
		{
			EventId:           "RPG26ND200002",
			GenconTicketId:    "TXN200-1",
			PurchaserGenconId: "GC1001",
			PurchaserName:     "Party Leader",
			RecipientGenconId: "GC1004",
			RecipientName:     "Solo User",
			TicketType:        "physical",
		},
	}
	tickets, _ := postgres.SyncPartyTickets(db, 101, 2026, "leader@example.com", ticketsInput)
	var pt *postgres.PartyTicket
	for _, t := range tickets {
		if t.GenconTicketId == "TXN200-1" {
			pt = t
		}
	}
	if pt.HolderEmail != "leader@example.com" {
		t.Fatalf("expected initial holder leader@example.com, got %s", pt.HolderEmail)
	}

	// 2. solo joins party 101!
	errJoin := postgres.JoinParty(db, 101, "solo@example.com")
	if errJoin != nil {
		t.Fatalf("JoinParty failed: %v", errJoin)
	}

	// 3. Verify ticket holder is now automatically solo@example.com!
	reloadedTickets, _ := postgres.LoadPartyTickets(db, 101, 2026)
	var matchedTicket *postgres.PartyTicket
	for _, t := range reloadedTickets {
		if t.TicketId == pt.TicketId {
			matchedTicket = t
		}
	}
	if matchedTicket.HolderEmail != "solo@example.com" {
		t.Errorf("expected matched holder solo@example.com, got %s", matchedTicket.HolderEmail)
	}
}

func TestSyncPartyTickets_MatchByPartyMemberOnly(t *testing.T) {
	repo := setupSeededDB(t)
	db := repo.DB

	// 1. Create a user with NO gencon_name or gencon_id
	_, err := db.Exec(`
		INSERT INTO users (email, display_name, gencon_name, gencon_id)
		VALUES ('chris_test@example.com', 'Chris Test', '', '')
		ON CONFLICT (email) DO UPDATE SET gencon_name = '', gencon_id = ''`)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	// 2. Add them to party 101, but ONLY set gencon_name in party_members table
	_, err = db.Exec(`
		INSERT INTO party_members (party_id, email, display_name, gencon_name, gencon_id)
		VALUES (101, 'chris_test@example.com', 'Chris', 'Christopher Parsons', '')
		ON CONFLICT (party_id, email) DO UPDATE SET gencon_name = 'Christopher Parsons', gencon_id = ''`)
	if err != nil {
		t.Fatalf("failed to insert party member: %v", err)
	}

	// 3. Sync a ticket where the recipient name is 'Christopher Parsons'
	ticketsInput := []postgres.TicketSyncInput{
		{
			EventId:           "RPG26ND200001",
			GenconTicketId:    "TXN999-1",
			PurchaserGenconId: "GC1001",
			PurchaserName:     "Party Leader",
			RecipientGenconId: "",
			RecipientName:     "Christopher Parsons",
			TicketType:        "physical",
		},
	}

	tickets, err := postgres.SyncPartyTickets(db, 101, 2026, "leader@example.com", ticketsInput)
	if err != nil {
		t.Fatalf("SyncPartyTickets failed: %v", err)
	}

	// Find our synced ticket
	var foundTicket *postgres.PartyTicket
	for _, pt := range tickets {
		if pt.GenconTicketId == "TXN999-1" {
			foundTicket = pt
		}
	}

	if foundTicket == nil {
		t.Fatalf("missing synced ticket TXN999-1")
	}

	// It should be mapped to 'chris_test@example.com' (Christopher Parsons)
	if foundTicket.HolderEmail != "chris_test@example.com" {
		t.Errorf("expected holder chris_test@example.com, got %s", foundTicket.HolderEmail)
	}
}
