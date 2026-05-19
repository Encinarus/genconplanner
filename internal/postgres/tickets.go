package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type PartyTicket struct {
	TicketId            string    `json:"ticketId"`
	PartyId             int64     `json:"partyId"`
	EventId             string    `json:"eventId"`
	Year                int       `json:"year"`
	PurchaserEmail      string    `json:"purchaserEmail"`
	GenconPurchaserName string    `json:"genconPurchaserName"`
	GenconTicketId      string    `json:"genconTicketId"`
	GenconRecipientName string    `json:"genconRecipientName"`
	GenconRecipientId   string    `json:"genconRecipientId"`
	HolderEmail         string    `json:"holderEmail"`
	HolderDisplayName   string    `json:"holderDisplayName"`
	TicketType          string    `json:"ticketType"`
	TicketStatus        string    `json:"ticketStatus"`
	TransferStatus      string    `json:"transferStatus"`
	CreatedAt           time.Time `json:"createdAt"`
	LastModified        time.Time `json:"lastModified"`
	// Enriched fields from events table
	EventTitle     string `json:"eventTitle,omitempty"`
	EventStartTime string `json:"eventStartTime,omitempty"`
	EventLocation  string `json:"eventLocation,omitempty"`
	EventCategory  string `json:"eventCategory,omitempty"`
}

type TicketTransfer struct {
	TransferId   string    `json:"transferId"`
	TicketId     string    `json:"ticketId"`
	PartyId      int64     `json:"partyId"`
	FromEmail    string    `json:"fromEmail"`
	ToEmail      string    `json:"toEmail"`
	TransferType string    `json:"transferType"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type TicketSyncInput struct {
	EventId           string `json:"eventId"`
	GenconTicketId    string `json:"genconTicketId"`
	PurchaserGenconId string `json:"purchaserGenconId"`
	PurchaserName     string `json:"purchaserName"`
	RecipientGenconId string `json:"recipientGenconId"`
	RecipientName     string `json:"recipientName"`
	TicketType        string `json:"ticketType"`
	Status            string `json:"status"` // "active" or "returned"
}

// SyncTicketHoldersToStarredEvents keeps the starred_events table in sync with ticket holders.
func SyncTicketHoldersToStarredEvents(db *sql.DB, holderEmail string, formerHolderEmail string, eventId string) error {
	if holderEmail != "" {
		_, err := UpdateStarredEventInternal(db, holderEmail, eventId, "purchased", false, true, false)
		if err != nil {
			return fmt.Errorf("failed to sync new holder starred event: %w", err)
		}
	}
	if formerHolderEmail != "" && formerHolderEmail != holderEmail {
		_, err := UpdateStarredEventInternal(db, formerHolderEmail, eventId, "must_have", false, true, false)
		if err != nil {
			return fmt.Errorf("failed to sync former holder starred event: %w", err)
		}
	}
	return nil
}

// resolvePartyMemberEmail attempts to match a Gen Con ID or Name to an active party member.
func resolvePartyMemberEmail(db *sql.DB, partyId int64, genconId string, genconName string, defaultEmail string) string {
	if genconId == "" && genconName == "" {
		return defaultEmail
	}

	query := `
SELECT u.email 
FROM users u
JOIN party_members pm ON u.email = pm.email
WHERE pm.party_id = $1 AND (u.gencon_id = $2 OR lower(u.gencon_name) = $3)
LIMIT 1`
	var email string
	err := db.QueryRow(query, partyId, genconId, strings.ToLower(genconName)).Scan(&email)
	if err == nil && email != "" {
		return email
	}
	return defaultEmail
}

// SyncPartyTickets batch-ingests tickets from the Chrome extension.
func SyncPartyTickets(db *sql.DB, partyId int64, year int, authEmail string, tickets []TicketSyncInput) ([]*PartyTicket, error) {
	authEmail = strings.ToLower(authEmail)

	// Process active purchases first
	for _, t := range tickets {
		if t.Status == "returned" {
			continue
		}

		purchaserEmail := resolvePartyMemberEmail(db, partyId, t.PurchaserGenconId, t.PurchaserName, authEmail)
		holderEmail := resolvePartyMemberEmail(db, partyId, t.RecipientGenconId, t.RecipientName, purchaserEmail)

		var existingTicketId string
		var formerHolderEmail string
		err := db.QueryRow(`
SELECT ticket_id, holder_email FROM party_tickets 
WHERE party_id = $1 AND year = $2 AND gencon_ticket_id = $3 AND event_id = $4
LIMIT 1`, partyId, year, t.GenconTicketId, t.EventId).Scan(&existingTicketId, &formerHolderEmail)

		if err == sql.ErrNoRows || existingTicketId == "" {
			// Insert new ticket
			var newTicketId string
			errInsert := db.QueryRow(`
INSERT INTO party_tickets (party_id, event_id, year, purchaser_email, gencon_purchaser_name, gencon_ticket_id, gencon_recipient_name, gencon_recipient_id, holder_email, ticket_type, ticket_status, transfer_status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'active', 'none')
RETURNING ticket_id`,
				partyId, t.EventId, year, purchaserEmail, t.PurchaserName, t.GenconTicketId, t.RecipientName, t.RecipientGenconId, holderEmail, t.TicketType).Scan(&newTicketId)
			if errInsert != nil {
				log.Printf("Error inserting synced ticket: %v", errInsert)
				continue
			}
			_ = SyncTicketHoldersToStarredEvents(db, holderEmail, "", t.EventId)
		} else if err == nil {
			// Update existing ticket if holder changed or recipient info updated
			_, errUpdate := db.Exec(`
UPDATE party_tickets 
SET gencon_purchaser_name = $1, gencon_recipient_name = $2, gencon_recipient_id = $3, holder_email = $4, last_modified = now()
WHERE ticket_id = $5`, t.PurchaserName, t.RecipientName, t.RecipientGenconId, holderEmail, existingTicketId)
			if errUpdate != nil {
				log.Printf("Error updating synced ticket: %v", errUpdate)
				continue
			}
			if holderEmail != formerHolderEmail {
				_ = SyncTicketHoldersToStarredEvents(db, holderEmail, formerHolderEmail, t.EventId)
			}
		}
	}

	// Process returned tickets second
	for _, t := range tickets {
		if t.Status != "returned" {
			continue
		}

		// Check if this return was already processed
		var alreadyReturnedId string
		errCheck := db.QueryRow("SELECT ticket_id FROM party_tickets WHERE party_id = $1 AND gencon_return_id = $2 LIMIT 1", partyId, t.GenconTicketId).Scan(&alreadyReturnedId)
		if errCheck == nil && alreadyReturnedId != "" {
			// Already processed this return row
			continue
		}

		// Find the latest active ticket for this recipient and event
		var ticketToReturnId string
		var holderEmail string
		errFind := db.QueryRow(`
SELECT ticket_id, holder_email FROM party_tickets 
WHERE party_id = $1 AND year = $2 AND event_id = $3 AND (lower(gencon_recipient_name) = $4 OR lower(holder_email) = $5) AND ticket_status = 'active'
ORDER BY created_at DESC 
LIMIT 1`, partyId, year, t.EventId, strings.ToLower(t.RecipientName), strings.ToLower(t.RecipientName)).Scan(&ticketToReturnId, &holderEmail)

		if errFind == nil && ticketToReturnId != "" {
			_, errUpdate := db.Exec("UPDATE party_tickets SET ticket_status = 'returned', gencon_return_id = $1, last_modified = now() WHERE ticket_id = $2", t.GenconTicketId, ticketToReturnId)
			if errUpdate == nil {
				// Remove the 'purchased' starred event for the holder
				_ = SyncTicketHoldersToStarredEvents(db, "", holderEmail, t.EventId)
			} else {
				log.Printf("Error marking ticket returned: %v", errUpdate)
			}
		}
	}

	return LoadPartyTickets(db, partyId, year)
}

// LoadPartyTickets retrieves all tickets for a party in a given year.
func LoadPartyTickets(db *sql.DB, partyId int64, year int) ([]*PartyTicket, error) {
	query := `
SELECT 
	pt.ticket_id, pt.party_id, pt.event_id, pt.year, pt.purchaser_email, COALESCE(pt.gencon_purchaser_name, ''), 
	COALESCE(pt.gencon_ticket_id, ''), pt.gencon_recipient_name, COALESCE(pt.gencon_recipient_id, ''), 
	pt.holder_email, u.display_name, pt.ticket_type, pt.ticket_status, pt.transfer_status, pt.created_at, pt.last_modified,
	COALESCE(e.title, ''), COALESCE(e.start_time, '1970-01-01 00:00:00-00'::timestamptz), COALESCE(e.location, ''), COALESCE(e.event_type, '')
FROM party_tickets pt
JOIN users u ON pt.holder_email = u.email
LEFT JOIN events e ON pt.event_id = e.event_id
WHERE pt.party_id = $1 AND pt.year = $2
ORDER BY e.start_time, pt.created_at`

	rows, err := db.Query(query, partyId, year)
	if err != nil {
		return nil, fmt.Errorf("failed to query party tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*PartyTicket
	for rows.Next() {
		var t PartyTicket
		var startTime time.Time
		if err := rows.Scan(
			&t.TicketId, &t.PartyId, &t.EventId, &t.Year, &t.PurchaserEmail, &t.GenconPurchaserName,
			&t.GenconTicketId, &t.GenconRecipientName, &t.GenconRecipientId,
			&t.HolderEmail, &t.HolderDisplayName, &t.TicketType, &t.TicketStatus, &t.TransferStatus,
			&t.CreatedAt, &t.LastModified, &t.EventTitle, &startTime, &t.EventLocation, &t.EventCategory,
		); err != nil {
			return nil, fmt.Errorf("failed to scan party ticket: %w", err)
		}
		t.EventStartTime = startTime.Format(time.RFC3339)
		tickets = append(tickets, &t)
	}
	return tickets, nil
}

// AddPartyTicket manually adds a ticket.
func AddPartyTicket(db *sql.DB, partyId int64, year int, eventId, purchaserEmail, genconRecipientName, holderEmail, ticketType string) (*PartyTicket, error) {
	purchaserEmail = strings.ToLower(purchaserEmail)
	holderEmail = strings.ToLower(holderEmail)

	var ticketId string
	err := db.QueryRow(`
INSERT INTO party_tickets (party_id, event_id, year, purchaser_email, gencon_purchaser_name, gencon_recipient_name, holder_email, ticket_type, ticket_status, transfer_status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', 'none')
RETURNING ticket_id`,
		partyId, eventId, year, purchaserEmail, purchaserEmail, genconRecipientName, holderEmail, ticketType).Scan(&ticketId)

	if err != nil {
		return nil, fmt.Errorf("failed to insert party ticket: %w", err)
	}

	_ = SyncTicketHoldersToStarredEvents(db, holderEmail, "", eventId)

	tickets, err := LoadPartyTickets(db, partyId, year)
	if err != nil {
		return nil, err
	}
	for _, t := range tickets {
		if t.TicketId == ticketId {
			return t, nil
		}
	}
	return nil, fmt.Errorf("ticket inserted but not found on reload")
}

// DeletePartyTicket deletes a ticket.
func DeletePartyTicket(db *sql.DB, partyId int64, ticketId string) error {
	var holderEmail, eventId string
	err := db.QueryRow("SELECT holder_email, event_id FROM party_tickets WHERE party_id = $1 AND ticket_id = $2", partyId, ticketId).Scan(&holderEmail, &eventId)
	if err == sql.ErrNoRows {
		return fmt.Errorf("ticket not found")
	} else if err != nil {
		return fmt.Errorf("failed to query ticket for deletion: %w", err)
	}

	_, err = db.Exec("DELETE FROM party_tickets WHERE party_id = $1 AND ticket_id = $2", partyId, ticketId)
	if err != nil {
		return fmt.Errorf("failed to delete ticket: %w", err)
	}

	_ = SyncTicketHoldersToStarredEvents(db, "", holderEmail, eventId)
	return nil
}

// InitiateTicketTransfer initiates a transfer between party members.
func InitiateTicketTransfer(db *sql.DB, partyId int64, ticketId, fromEmail, toEmail, transferType string) (*TicketTransfer, error) {
	fromEmail = strings.ToLower(fromEmail)
	toEmail = strings.ToLower(toEmail)

	var eventId string
	err := db.QueryRow("SELECT event_id FROM party_tickets WHERE party_id = $1 AND ticket_id = $2 AND holder_email = $3", partyId, ticketId, fromEmail).Scan(&eventId)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ticket not found or not held by sender")
	} else if err != nil {
		return nil, fmt.Errorf("failed to verify ticket ownership: %w", err)
	}

	status := "pending"
	if transferType == "name_only" {
		status = "completed"
	}

	var transferId string
	errInsert := db.QueryRow(`
INSERT INTO ticket_transfers (ticket_id, party_id, from_email, to_email, transfer_type, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING transfer_id`, ticketId, partyId, fromEmail, toEmail, transferType, status).Scan(&transferId)

	if errInsert != nil {
		return nil, fmt.Errorf("failed to insert ticket transfer: %w", errInsert)
	}

	if transferType == "name_only" {
		_, errUpdate := db.Exec("UPDATE party_tickets SET holder_email = $1, transfer_status = 'name_only_transfer', last_modified = now() WHERE ticket_id = $2", toEmail, ticketId)
		if errUpdate != nil {
			return nil, fmt.Errorf("failed to update ticket holder: %w", errUpdate)
		}
		_ = SyncTicketHoldersToStarredEvents(db, toEmail, fromEmail, eventId)
	} else {
		_, errUpdate := db.Exec("UPDATE party_tickets SET transfer_status = 'pending_gencon_transfer', last_modified = now() WHERE ticket_id = $1", ticketId)
		if errUpdate != nil {
			return nil, fmt.Errorf("failed to update ticket transfer status: %w", errUpdate)
		}
	}

	return getTicketTransfer(db, transferId)
}

// RespondTicketTransfer handles accept/reject/complete actions for pending e-ticket transfers.
func RespondTicketTransfer(db *sql.DB, partyId int64, transferId, action string) (*TicketTransfer, error) {
	var ticketId, fromEmail, toEmail, status, eventId string
	err := db.QueryRow(`
SELECT tt.ticket_id, tt.from_email, tt.to_email, tt.status, pt.event_id 
FROM ticket_transfers tt
JOIN party_tickets pt ON tt.ticket_id = pt.ticket_id
WHERE tt.party_id = $1 AND tt.transfer_id = $2`, partyId, transferId).Scan(&ticketId, &fromEmail, &toEmail, &status, &eventId)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transfer request not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to query transfer: %w", err)
	}

	if status != "pending" {
		return nil, fmt.Errorf("transfer is not in pending state")
	}

	newStatus := status
	if action == "accept" || action == "complete" {
		newStatus = "completed"
		_, errUpdate := db.Exec("UPDATE party_tickets SET holder_email = $1, transfer_status = 'completed', last_modified = now() WHERE ticket_id = $2", toEmail, ticketId)
		if errUpdate != nil {
			return nil, fmt.Errorf("failed to update ticket holder: %w", errUpdate)
		}
		_ = SyncTicketHoldersToStarredEvents(db, toEmail, fromEmail, eventId)
	} else if action == "reject" {
		newStatus = "rejected"
		_, errUpdate := db.Exec("UPDATE party_tickets SET transfer_status = 'none', last_modified = now() WHERE ticket_id = $1", ticketId)
		if errUpdate != nil {
			return nil, fmt.Errorf("failed to reset ticket transfer status: %w", errUpdate)
		}
	} else {
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	_, errUpdateTransfer := db.Exec("UPDATE ticket_transfers SET status = $1, updated_at = now() WHERE transfer_id = $2", newStatus, transferId)
	if errUpdateTransfer != nil {
		return nil, fmt.Errorf("failed to update transfer status: %w", errUpdateTransfer)
	}

	return getTicketTransfer(db, transferId)
}

func getTicketTransfer(db *sql.DB, transferId string) (*TicketTransfer, error) {
	var tt TicketTransfer
	err := db.QueryRow(`
SELECT transfer_id, ticket_id, party_id, from_email, to_email, transfer_type, status, created_at, updated_at
FROM ticket_transfers WHERE transfer_id = $1`, transferId).Scan(
		&tt.TransferId, &tt.TicketId, &tt.PartyId, &tt.FromEmail, &tt.ToEmail,
		&tt.TransferType, &tt.Status, &tt.CreatedAt, &tt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tt, nil
}

// MatchPendingPartyTickets scans existing party tickets and reassigns unmapped tickets when new members join or update profiles.
func MatchPendingPartyTickets(db *sql.DB, partyId int64, year int) error {
	// Find all members of the party and their Gen Con IDs / Names
	rows, err := db.Query(`
SELECT u.email, COALESCE(u.gencon_id, ''), COALESCE(u.gencon_name, '')
FROM users u
JOIN party_members pm ON u.email = pm.email
WHERE pm.party_id = $1`, partyId)
	if err != nil {
		return fmt.Errorf("failed to query party members for matching: %w", err)
	}
	defer rows.Close()

	type memberInfo struct {
		email      string
		genconId   string
		genconName string
	}
	var members []memberInfo
	for rows.Next() {
		var m memberInfo
		if err := rows.Scan(&m.email, &m.genconId, &m.genconName); err != nil {
			return fmt.Errorf("failed to scan member info: %w", err)
		}
		members = append(members, m)
	}
	rows.Close()

	// For each member, find tickets where gencon_recipient_id or gencon_recipient_name matches, but holder_email does not match
	for _, m := range members {
		if m.genconId == "" && m.genconName == "" {
			continue
		}

		query := `
SELECT ticket_id, holder_email, event_id 
FROM party_tickets 
WHERE party_id = $1 AND year = $2 AND holder_email != $3 
  AND (gencon_recipient_id = $4 OR (length($5) > 0 AND lower(gencon_recipient_name) = $5))`

		ticketRows, errT := db.Query(query, partyId, year, m.email, m.genconId, strings.ToLower(m.genconName))
		if errT != nil {
			log.Printf("Error querying pending tickets for %s: %v", m.email, errT)
			continue
		}

		type pendingTicket struct {
			ticketId          string
			formerHolderEmail string
			eventId           string
		}
		var pTickets []pendingTicket
		for ticketRows.Next() {
			var pt pendingTicket
			if err := ticketRows.Scan(&pt.ticketId, &pt.formerHolderEmail, &pt.eventId); err == nil {
				pTickets = append(pTickets, pt)
			}
		}
		ticketRows.Close()

		for _, pt := range pTickets {
			_, errU := db.Exec("UPDATE party_tickets SET holder_email = $1, last_modified = now() WHERE ticket_id = $2", m.email, pt.ticketId)
			if errU == nil {
				_ = SyncTicketHoldersToStarredEvents(db, m.email, pt.formerHolderEmail, pt.eventId)
			}
		}
	}

	return nil
}

// ToggleTicketReturn toggles a ticket between 'active' and 'returned'.
func ToggleTicketReturn(db *sql.DB, partyId int64, ticketId string) (*PartyTicket, error) {
	var currentStatus, holderEmail, eventId string
	err := db.QueryRow("SELECT ticket_status, holder_email, event_id FROM party_tickets WHERE party_id = $1 AND ticket_id = $2", partyId, ticketId).Scan(&currentStatus, &holderEmail, &eventId)
	if err != nil {
		return nil, fmt.Errorf("failed to find ticket: %w", err)
	}

	newStatus := "returned"
	if currentStatus == "returned" {
		newStatus = "active"
	}

	_, errUpdate := db.Exec("UPDATE party_tickets SET ticket_status = $1, last_modified = now() WHERE party_id = $2 AND ticket_id = $3", newStatus, partyId, ticketId)
	if errUpdate != nil {
		return nil, fmt.Errorf("failed to update ticket status: %w", errUpdate)
	}

	if newStatus == "returned" {
		_ = SyncTicketHoldersToStarredEvents(db, "", holderEmail, eventId)
	} else {
		_ = SyncTicketHoldersToStarredEvents(db, holderEmail, "", eventId)
	}

	var t PartyTicket
	var startTime time.Time
	errReload := db.QueryRow(`
SELECT 
	pt.ticket_id, pt.party_id, pt.event_id, pt.year, pt.purchaser_email, COALESCE(pt.gencon_purchaser_name, ''), 
	COALESCE(pt.gencon_ticket_id, ''), pt.gencon_recipient_name, COALESCE(pt.gencon_recipient_id, ''), 
	pt.holder_email, u.display_name, pt.ticket_type, pt.ticket_status, pt.transfer_status, pt.created_at, pt.last_modified,
	COALESCE(e.title, ''), COALESCE(e.start_time, '1970-01-01 00:00:00-00'::timestamptz), COALESCE(e.location, ''), COALESCE(e.event_type, '')
FROM party_tickets pt
JOIN users u ON pt.holder_email = u.email
LEFT JOIN events e ON pt.event_id = e.event_id
WHERE pt.party_id = $1 AND pt.ticket_id = $2`, partyId, ticketId).Scan(
		&t.TicketId, &t.PartyId, &t.EventId, &t.Year, &t.PurchaserEmail, &t.GenconPurchaserName,
		&t.GenconTicketId, &t.GenconRecipientName, &t.GenconRecipientId,
		&t.HolderEmail, &t.HolderDisplayName, &t.TicketType, &t.TicketStatus, &t.TransferStatus,
		&t.CreatedAt, &t.LastModified, &t.EventTitle, &startTime, &t.EventLocation, &t.EventCategory,
	)
	if errReload != nil {
		return nil, fmt.Errorf("failed to reload ticket: %w", errReload)
	}
	t.EventStartTime = startTime.Format(time.RFC3339)
	return &t, nil
}
