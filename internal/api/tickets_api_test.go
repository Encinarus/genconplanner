package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Encinarus/genconplanner/internal/postgres"
)

func TestSyncTickets_API(t *testing.T) {
	server, stub, auth, _, router := setupTestServer()
	server.RegisterRoutes(router.Group("/api"))

	auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
		return "leader@example.com", nil
	}

	stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
		return &postgres.User{Email: email}, nil
	}

	stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
		return []*postgres.Party{{Id: 101, Year: 2026}}, nil
	}

	stub.SyncPartyTicketsFn = func(partyId int64, year int, authEmail string, tickets []postgres.TicketSyncInput) ([]*postgres.PartyTicket, error) {
		return []*postgres.PartyTicket{{TicketId: "t1", HolderEmail: "leader@example.com"}}, nil
	}

	body := SyncTicketsRequest{
		Source: "chrome_extension",
		Tickets: []postgres.TicketSyncInput{
			{EventId: "BGM26ND100001", GenconTicketId: "TXN100-1"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/party/2026/tickets/sync", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetTickets_API(t *testing.T) {
	server, stub, auth, _, router := setupTestServer()
	server.RegisterRoutes(router.Group("/api"))

	auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
		return "leader@example.com", nil
	}

	stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
		return &postgres.User{Email: email}, nil
	}

	stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
		return []*postgres.Party{{Id: 101, Year: 2026}}, nil
	}

	stub.LoadPartyTicketsFn = func(partyId int64, year int) ([]*postgres.PartyTicket, error) {
		return []*postgres.PartyTicket{{TicketId: "t1", HolderEmail: "leader@example.com"}}, nil
	}

	req, _ := http.NewRequest("GET", "/api/v1/party/2026/tickets", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestTransferTicket_API(t *testing.T) {
	server, stub, auth, _, router := setupTestServer()
	server.RegisterRoutes(router.Group("/api"))

	auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
		return "leader@example.com", nil
	}

	stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
		return &postgres.User{Email: email}, nil
	}

	stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
		return []*postgres.Party{{Id: 101, Year: 2026}}, nil
	}

	stub.InitiateTicketTransferFn = func(partyId int64, ticketId, fromEmail, toEmail, transferType string) (*postgres.TicketTransfer, error) {
		return &postgres.TicketTransfer{TransferId: "tr1", Status: "completed"}, nil
	}

	body := TransferTicketRequest{
		ToEmail:      "member1@example.com",
		TransferType: "name_only",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/party/2026/tickets/t1/transfer", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestRespondTransfer_API_Authorization(t *testing.T) {
	server, stub, auth, _, router := setupTestServer()
	server.RegisterRoutes(router.Group("/api"))

	auth.VerifyIDTokenFn = func(ctx context.Context, token string) (string, error) {
		return "attacker@example.com", nil
	}

	stub.LoadOrCreateUserFn = func(email string) (*postgres.User, error) {
		return &postgres.User{Email: email}, nil
	}

	stub.LoadPartiesFn = func(user *postgres.User) ([]*postgres.Party, error) {
		return []*postgres.Party{{Id: 101, Year: 2026}}, nil
	}

	stub.RespondTicketTransferFn = func(partyId int64, transferId, action, callerEmail string) (*postgres.TicketTransfer, error) {
		if callerEmail != "attacker@example.com" {
			return nil, fmt.Errorf("unexpected caller email: %s", callerEmail)
		}
		// Return an error to simulate unauthorized rejection/accept check inside DB layer
		return nil, fmt.Errorf("unauthorized to respond to this transfer")
	}

	body := RespondTransferRequest{
		Action: "accept",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/party/2026/transfers/tr1/respond", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d. Body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "unauthorized") {
		t.Fatalf("expected unauthorized error message, got: %s", w.Body.String())
	}
}

