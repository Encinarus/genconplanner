package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

func (s *Server) registerTicketRoutes(group *gin.RouterGroup) {
	group.POST("/party/:party_id/tickets/sync", s.SyncTickets)
	group.GET("/party/:party_id/tickets", s.GetTickets)
	group.POST("/party/:party_id/tickets", s.AddTicket)
	group.DELETE("/party/:party_id/tickets/:ticket_id", s.DeleteTicket)
	group.POST("/party/:party_id/tickets/:ticket_id/transfer", s.TransferTicket)
	group.POST("/party/:party_id/transfers/:transfer_id/respond", s.RespondTransfer)
	group.POST("/party/:party_id/tickets/:ticket_id/toggle_return", s.ToggleTicketReturn)
}

func (s *Server) getPartyForYear(yearParam string, email string) (*postgres.Party, int, error) {
	year, err := strconv.Atoi(yearParam)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid year parameter")
	}

	user, err := s.Repo.LoadOrCreateUser(email)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load user: %w", err)
	}

	parties, err := s.Repo.LoadParties(user)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load parties: %w", err)
	}

	for _, p := range parties {
		if p.Year == int64(year) {
			return p, year, nil
		}
	}

	return nil, year, fmt.Errorf("user does not belong to a party for year %d", year)
}

type SyncTicketsRequest struct {
	Source  string                     `json:"source"`
	Tickets []postgres.TicketSyncInput `json:"tickets"`
}

func (s *Server) SyncTickets(c *gin.Context) {
	email := GetUserEmail(c)
	party, year, err := s.getPartyForYear(c.Param("party_id"), email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	var req SyncTicketsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload"})
		return
	}

	tickets, errSync := s.Repo.SyncPartyTickets(party.Id, year, email, req.Tickets)
	if errSync != nil {
		log.Printf("SyncTickets error: %v\n", errSync)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to sync tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"syncedCount": len(req.Tickets),
		"tickets":     tickets,
	})
}

func (s *Server) GetTickets(c *gin.Context) {
	email := GetUserEmail(c)
	party, year, err := s.getPartyForYear(c.Param("party_id"), email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	tickets, errLoad := s.Repo.LoadPartyTickets(party.Id, year)
	if errLoad != nil {
		log.Printf("GetTickets error: %v\n", errLoad)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to load tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"tickets": tickets,
	})
}

type AddTicketRequest struct {
	EventId             string `json:"eventId" binding:"required"`
	PurchaserEmail      string `json:"purchaserEmail" binding:"required"`
	GenconRecipientName string `json:"genconRecipientName" binding:"required"`
	HolderEmail         string `json:"holderEmail" binding:"required"`
	TicketType          string `json:"ticketType" binding:"required"`
}

func (s *Server) AddTicket(c *gin.Context) {
	email := GetUserEmail(c)
	party, year, err := s.getPartyForYear(c.Param("party_id"), email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	var req AddTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload"})
		return
	}

	ticket, errAdd := s.Repo.AddPartyTicket(party.Id, year, req.EventId, req.PurchaserEmail, req.GenconRecipientName, req.HolderEmail, req.TicketType)
	if errAdd != nil {
		log.Printf("AddTicket error: %v\n", errAdd)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to add ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"ticket": ticket,
	})
}

func (s *Server) DeleteTicket(c *gin.Context) {
	email := GetUserEmail(c)
	party, _, err := s.getPartyForYear(c.Param("party_id"), email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	ticketId := c.Param("ticket_id")
	if errDel := s.Repo.DeletePartyTicket(party.Id, ticketId); errDel != nil {
		log.Printf("DeleteTicket error: %v\n", errDel)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

type TransferTicketRequest struct {
	ToEmail      string `json:"toEmail" binding:"required"`
	TransferType string `json:"transferType" binding:"required"`
	Notes        string `json:"notes"`
}

func (s *Server) TransferTicket(c *gin.Context) {
	email := GetUserEmail(c)
	party, _, err := s.getPartyForYear(c.Param("party_id"), email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	ticketId := c.Param("ticket_id")
	var req TransferTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload"})
		return
	}

	transfer, errTrans := s.Repo.InitiateTicketTransfer(party.Id, ticketId, email, req.ToEmail, req.TransferType)
	if errTrans != nil {
		log.Printf("TransferTicket error: %v\n", errTrans)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to initiate transfer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"transfer": transfer,
	})
}

type RespondTransferRequest struct {
	Action string `json:"action" binding:"required"`
}

func (s *Server) RespondTransfer(c *gin.Context) {
	email := GetUserEmail(c)
	party, _, err := s.getPartyForYear(c.Param("party_id"), email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	transferId := c.Param("transfer_id")
	var req RespondTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request payload"})
		return
	}

	transfer, errResp := s.Repo.RespondTicketTransfer(party.Id, transferId, req.Action, email)
	if errResp != nil {
		log.Printf("RespondTransfer error: %v\n", errResp)
		if strings.Contains(errResp.Error(), "unauthorized") {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: errResp.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to respond to transfer"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"transfer": transfer,
	})
}

func (s *Server) ToggleTicketReturn(c *gin.Context) {
	email := GetUserEmail(c)
	party, _, err := s.getPartyForYear(c.Param("party_id"), email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	ticketId := c.Param("ticket_id")
	ticket, errToggle := s.Repo.ToggleTicketReturn(party.Id, ticketId)
	if errToggle != nil {
		log.Printf("ToggleTicketReturn error: %v\n", errToggle)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to toggle ticket return status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"ticket": ticket,
	})
}
