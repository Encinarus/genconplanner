package api

import (
	"log"
	"net/http"
	"regexp"
	"strconv"
	"fmt"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/Encinarus/genconplanner/internal/prioritization"
	"github.com/gin-gonic/gin"
)

type User struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type UserEvents struct {
	Email           string            `json:"email"`
	Year            int               `json:"year"`
	StarredClusters []string          `json:"starredClusters"`
	StarredEvents   []string          `json:"starredEvents"`
	StarredTiers    map[string]string `json:"starredTiers"`
	TicketedEvents  []string          `json:"ticketedEvents"`
}

type StarredEventDetail struct {
	EventId          string `json:"eventId"`
	Title            string `json:"title"`
	ShortDescription string `json:"shortDescription"`
	CategoryCode     string `json:"categoryCode"`
	StartTime        string `json:"startTime"`
	EndTime          string `json:"endTime"`
	GenconUrl        string `json:"genconUrl"`
	PlannerUrl       string `json:"plannerUrl"`
	Tier             string `json:"tier"`
}

type StarredPageData struct {
	Email            string                 `json:"email"`
	Year             int                    `json:"year"`
	CalendarEvents   []CalendarEventCluster `json:"calendarEvents"`
	IndividualEvents []StarredEventDetail   `json:"individualEvents"`
	Metadata         CalendarMetadata       `json:"metadata"`
	StarredClusters  []string               `json:"starredClusters"`
	StarredEvents    []string               `json:"starredEvents"`
}

type WishlistItem struct {
	Event     StarredEventDetail `json:"event"`
	Status    string             `json:"status"`
	Reasoning []string           `json:"reasoning"`
	Score     float64            `json:"score"`
}

type Party struct {
	Id          int64         `json:"id"`
	Name        string        `json:"name"`
	Year        int64         `json:"year"`
	LeaderEmail string        `json:"leaderEmail"`
	Members     []PartyMember `json:"members"`
}

type PartyMember struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

type WishlistConstraint struct {
	DayOfWeek          int `json:"dayOfWeek"`
	StartHour          int `json:"startHour"`
	StartMinute        int `json:"startMinute"`
	EndHour            int `json:"endHour"`
	EndMinute          int `json:"endMinute"`
	MinDurationMinutes int `json:"minDurationMinutes"`
}

func (s *Server) GetUser(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	dbUser, err := s.Repo.LoadOrCreateUser(email)
	if err != nil {
		log.Printf("error loading/creating user: %v\n", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, ErrorResponse{Error: "Service Unavailable"})
		return
	}

	var user User
	user.DisplayName = dbUser.DisplayName
	user.Email = dbUser.Email
	c.JSON(http.StatusOK, user)
}

func (s *Server) LoadUserEvents(c *gin.Context) {
	authedEmail := GetUserEmail(c)
	paramEmail := c.Param("email")
	yearParam := c.Param("year")

	if authedEmail == "" || authedEmail != paramEmail {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	var userEvents UserEvents
	userEvents.Email = authedEmail
	userEvents.Year = year
	userEvents.StarredClusters = make([]string, 0)
	userEvents.StarredEvents = make([]string, 0)
	userEvents.StarredTiers = make(map[string]string)
	userEvents.TicketedEvents = make([]string, 0)

	starredIds, err := s.Repo.GetStarredIds(authedEmail, year)
	if err != nil {
		log.Printf("error getting user starred list: %v\n", err)
	} else {
		for _, starred := range starredIds.StarredEvents {
			userEvents.StarredTiers[starred.EventId] = starred.Tier
			if starred.Level == "group" {
				userEvents.StarredClusters = append(userEvents.StarredClusters, starred.EventId)
			} else if starred.Level == "event" {
				userEvents.StarredEvents = append(userEvents.StarredEvents, starred.EventId)
			}
		}
	}

	c.JSON(http.StatusOK, userEvents)
}

type StarEventRequest struct {
	EventId string `json:"eventId"`
	Add     bool   `json:"add"`
	Related bool   `json:"related"`
	Tier    string `json:"tier"`
}

func (s *Server) StarEvent(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req StarEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	starred, err := s.Repo.UpdateStarredEventMinimal(email, req.EventId, req.Tier, req.Related, req.Add)
	if err != nil {
		log.Printf("error updating starred event: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, starred)
}

func (s *Server) GetStarredEvents(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	// Load event groups (clusters) to match legacy UI behavior
	log.Printf("Loading starred event groups for %s year %d\n", email, year)
	dbGroups, err := s.Repo.LoadStarredEventGroups(email, year)
	if err != nil {
		log.Printf("error loading starred event groups: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	log.Printf("Found %d starred event groups\n", len(dbGroups))

	apiResults := make([]EventSummary, 0)
	for _, dbGroup := range dbGroups {
		var summary EventSummary
		summary.AnchorEventId = dbGroup.EventId
		summary.Title = dbGroup.Name
		summary.ShortDescription = dbGroup.Description
		summary.NumEvents = dbGroup.Count
		summary.GameSystem.Name = dbGroup.GameSystem
		
		summary.WedTickets = dbGroup.WedTickets
		summary.ThuTickets = dbGroup.ThursTickets
		summary.FriTickets = dbGroup.FriTickets
		summary.SatTickets = dbGroup.SatTickets
		summary.SunTickets = dbGroup.SunTickets
		
		apiResults = append(apiResults, summary)
	}

	c.JSON(http.StatusOK, apiResults)
}

func (s *Server) GetStarredCalendarEvents(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	starredEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	clusters, err := s.Repo.LoadStarredEventClusters(email, year, starredEvents)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	apiClusters := make([]CalendarEventCluster, 0, len(clusters))
	for _, cluster := range clusters {
		apiClusters = append(apiClusters, CalendarEventCluster{
			EventId:          cluster.EventId,
			Title:            cluster.Title,
			StartTime:        cluster.StartTime,
			EndTime:          cluster.EndTime,
			GenconUrl:        cluster.GenconUrl,
			PlannerUrl:       cluster.PlannerUrl,
			ShortCategory:    cluster.ShortCategory,
			ShortDescription: cluster.ShortDescription,
			SimilarCount:     cluster.SimilarCount,
		})
	}

	c.JSON(http.StatusOK, apiClusters)
}

func (s *Server) GetCalendarMetadata(c *gin.Context) {
	yearParam := c.Param("year")
	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	c.JSON(http.StatusOK, CalendarMetadata{
		StartDate: events.GenconStartDate(year),
		EndDate:   events.GenconEndDate(year),
	})
}

func (s *Server) GetStarredIndividualEvents(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	dbEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		log.Printf("error loading starred individual events: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	results := make([]StarredEventDetail, 0)
	starredIds, _ := s.Repo.GetStarredIds(email, year)
	tiers := make(map[string]string)
	if starredIds != nil {
		for _, s := range starredIds.StarredEvents {
			tiers[s.EventId] = s.Tier
		}
	}

	for _, e := range dbEvents {
		results = append(results, StarredEventDetail{
			EventId:          e.EventId,
			Title:            e.Title,
			ShortDescription: e.ShortDescription,
			CategoryCode:     e.ShortCategory,
			StartTime:        e.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			EndTime:          e.EndTime.Format("2006-01-02T15:04:05Z07:00"),
			GenconUrl:        e.GenconLink(),
			PlannerUrl:       e.PlannerLink(),
			Tier:             tiers[e.EventId],
		})
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) GetStarredPageData(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	// 1. Get starred events (needed for both clusters and details)
	dbEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		log.Printf("error loading starred events: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// 2. Get clusters for calendar
	clusters, err := s.Repo.LoadStarredEventClusters(email, year, dbEvents)
	if err != nil {
		log.Printf("error loading clusters: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// 3. Get Starred IDs for global state sync
	starredIds, err := s.Repo.GetStarredIds(email, year)
	if err != nil {
		log.Printf("error getting starred ids: %v\n", err)
	}

	var data StarredPageData
	data.Email = email
	data.Year = year
	data.Metadata = CalendarMetadata{
		StartDate: events.GenconStartDate(year),
		EndDate:   events.GenconEndDate(year),
	}
	data.IndividualEvents = make([]StarredEventDetail, 0)
	data.CalendarEvents = make([]CalendarEventCluster, 0)
	data.StarredClusters = make([]string, 0)
	data.StarredEvents = make([]string, 0)

	for _, e := range dbEvents {
		tier := ""
		if starredIds != nil {
			for _, s := range starredIds.StarredEvents {
				if s.EventId == e.EventId {
					tier = s.Tier
					break
				}
			}
		}

		data.IndividualEvents = append(data.IndividualEvents, StarredEventDetail{
			EventId:          e.EventId,
			Title:            e.Title,
			ShortDescription: e.ShortDescription,
			CategoryCode:     e.ShortCategory,
			StartTime:        e.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			EndTime:          e.EndTime.Format("2006-01-02T15:04:05Z07:00"),
			GenconUrl:        e.GenconLink(),
			PlannerUrl:       e.PlannerLink(),
			Tier:             tier,
		})
	}

	for _, cluster := range clusters {
		data.CalendarEvents = append(data.CalendarEvents, CalendarEventCluster{
			EventId:          cluster.EventId,
			Title:            cluster.Title,
			StartTime:        cluster.StartTime,
			EndTime:          cluster.EndTime,
			GenconUrl:        cluster.GenconUrl,
			PlannerUrl:       cluster.PlannerUrl,
			ShortCategory:    cluster.ShortCategory,
			ShortDescription: cluster.ShortDescription,
			SimilarCount:     cluster.SimilarCount,
		})
	}

	if starredIds != nil {
		for _, s := range starredIds.StarredEvents {
			if s.Level == "group" {
				data.StarredClusters = append(data.StarredClusters, s.EventId)
			} else {
				data.StarredEvents = append(data.StarredEvents, s.EventId)
			}
		}
	}

	c.JSON(http.StatusOK, data)
}

func (s *Server) BulkClearStarredEvents(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	err = s.Repo.ClearStarredEvents(email, year)
	if err != nil {
		log.Printf("error clearing starred events: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

type BulkReplaceRequest struct {
	Text      string `json:"text"`
	Overwrite bool   `json:"overwrite"`
	AsGroups  bool   `json:"asGroups"`
}

func (s *Server) BulkReplaceStarredEvents(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	var req BulkReplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	// 1. Regex to find all event IDs
	// General regex for GenCon IDs: [A-Z]{3,4}\d{2}ND\d{6,}
	idRegex := regexp.MustCompile(`[A-Z]{3,4}\d{2}ND\d{6,}`)
	matches := idRegex.FindAllString(req.Text, -1)

	if len(matches) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "No valid event IDs found"})
		return
	}

	// 2. Validate that all IDs match the requested year
	yearLastTwo := fmt.Sprintf("%02d", year%100)
	var validIds []string
	seen := make(map[string]bool)

	for _, id := range matches {
		// Check the year part of the ID (e.g., BGM26ND... -> 26)
		// Prefix is 3 or 4 chars
		idYearPart := ""
		if id[3] >= '0' && id[3] <= '9' {
			idYearPart = id[3:5]
		} else {
			idYearPart = id[4:6]
		}

		if idYearPart != yearLastTwo {
			c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("Event ID %s is for a different year", id)})
			return
		}

		if !seen[id] {
			validIds = append(validIds, id)
			seen[id] = true
		}
	}

	err = s.Repo.BulkStarEvents(email, year, validIds, req.Overwrite, req.AsGroups)
	if err != nil {
		log.Printf("error bulk replacing starred events: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) GetAgenda(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	dbAgenda, err := s.Repo.LoadAgenda(email, year)
	if err != nil {
		log.Printf("error loading agenda: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	results := make([]StarredEventDetail, 0)
	for _, entry := range dbAgenda {
		results = append(results, StarredEventDetail{
			EventId:          entry.Event.EventId,
			Title:            entry.Event.Title,
			ShortDescription: entry.Event.ShortDescription,
			CategoryCode:     entry.Event.ShortCategory,
			StartTime:        entry.Event.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			EndTime:          entry.Event.EndTime.Format("2006-01-02T15:04:05Z07:00"),
			GenconUrl:        entry.Event.GenconLink(),
			PlannerUrl:       entry.Event.PlannerLink(),
			Tier:             entry.Tier,
		})
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) GetParties(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	user, err := s.Repo.LoadOrCreateUser(email)
	if err != nil {
		log.Printf("error loading user for parties: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	dbParties, err := s.Repo.LoadParties(user)
	if err != nil {
		log.Printf("error loading parties: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	apiParties := make([]Party, 0, len(dbParties))
	for _, p := range dbParties {
		members := make([]PartyMember, 0, len(p.Members))
		for _, m := range p.Members {
			members = append(members, PartyMember{
				DisplayName: m.DisplayName,
				Email:       m.Email,
			})
		}
		apiParties = append(apiParties, Party{
			Id:      p.Id,
			Name:    p.Name,
			Year:    p.Year,
			Members: members,
		})
	}

	c.JSON(http.StatusOK, apiParties)
}

type CreatePartyRequest struct {
	Name string `json:"name"`
	Year int64  `json:"year"`
}

func (s *Server) CreateParty(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req CreatePartyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	if req.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Party name is required"})
		return
	}

	dbParty, err := s.Repo.NewParty(req.Name, req.Year, email)
	if err != nil {
		log.Printf("error creating party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	members := make([]PartyMember, 0, len(dbParty.Members))
	for _, m := range dbParty.Members {
		members = append(members, PartyMember{
			DisplayName: m.DisplayName,
			Email:       m.Email,
		})
	}

	apiParty := Party{
		Id:          dbParty.Id,
		Name:        dbParty.Name,
		Year:        dbParty.Year,
		LeaderEmail: dbParty.LeaderEmail,
		Members:     members,
	}

	c.JSON(http.StatusOK, apiParty)
}

func (s *Server) GetParty(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("party_id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid party ID"})
		return
	}

	dbParty, err := s.Repo.LoadParty(id)
	if err != nil {
		log.Printf("error loading party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	members := make([]PartyMember, 0, len(dbParty.Members))
	for _, m := range dbParty.Members {
		members = append(members, PartyMember{
			DisplayName: m.DisplayName,
			Email:       m.Email,
		})
	}

	apiParty := Party{
		Id:          dbParty.Id,
		Name:        dbParty.Name,
		Year:        dbParty.Year,
		LeaderEmail: dbParty.LeaderEmail,
		Members:     members,
	}

	c.JSON(http.StatusOK, apiParty)
}

type RenamePartyRequest struct {
	Name string `json:"name"`
}

func (s *Server) RenameParty(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("party_id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid party ID"})
		return
	}

	var req RenamePartyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	// Auth check: must be leader
	party, err := s.Repo.LoadParty(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	if party.LeaderEmail != email {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Only the Party Leader can rename the party"})
		return
	}

	err = s.Repo.RenameParty(id, req.Name)
	if err != nil {
		log.Printf("error renaming party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type TransferLeadershipRequest struct {
	NewLeaderEmail string `json:"newLeaderEmail"`
}

func (s *Server) TransferLeadership(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("party_id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid party ID"})
		return
	}

	var req TransferLeadershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	// Auth check: must be leader
	party, err := s.Repo.LoadParty(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	if party.LeaderEmail != email {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Only the Party Leader can transfer leadership"})
		return
	}

	// Verify new leader is a member
	isMember := false
	for _, m := range party.Members {
		if m.Email == req.NewLeaderEmail {
			isMember = true
			break
		}
	}

	if !isMember {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "New leader must be a member of the party"})
		return
	}

	err = s.Repo.UpdatePartyLeader(id, req.NewLeaderEmail)
	if err != nil {
		log.Printf("error transferring leadership: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) JoinParty(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("party_id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid party ID"})
		return
	}

	err = s.Repo.JoinParty(id, email)
	if err != nil {
		log.Printf("error joining party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) LeaveParty(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("party_id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid party ID"})
		return
	}

	party, err := s.Repo.LoadParty(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	if party.LeaderEmail == email && len(party.Members) > 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Party Leader cannot leave unless they transfer leadership or are the last member"})
		return
	}

	err = s.Repo.RemoveMember(id, email)
	if err != nil {
		log.Printf("error leaving party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// If last member leaves, delete the party
	if len(party.Members) == 1 {
		err = s.Repo.DeleteParty(id)
		if err != nil {
			log.Printf("error deleting empty party: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) DeleteParty(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	id, err := strconv.ParseInt(c.Param("party_id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid party ID"})
		return
	}

	party, err := s.Repo.LoadParty(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	if party.LeaderEmail != email {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Only the Party Leader can delete the party"})
		return
	}

	if len(party.Members) > 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Party can only be deleted if there is only one member remaining"})
		return
	}

	err = s.Repo.DeleteParty(id)
	if err != nil {
		log.Printf("error deleting party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type RenameUserRequest struct {
	DisplayName string `json:"displayName"`
}

func (s *Server) RenameUser(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req RenameUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	if req.DisplayName == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Display name cannot be empty"})
		return
	}

	err := s.Repo.UpdateDisplayName(email, req.DisplayName)
	if err != nil {
		log.Printf("error renaming user: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) GetWishlist(c *gin.Context) {
	email := GetUserEmail(c)
	yearParam := c.Param("year")

	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid year"})
		return
	}

	starred, err := s.Repo.GetStarredIds(email, year)
	if err != nil {
		log.Printf("error loading starred ids: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	cache, dirty, err := s.Repo.GetWishlistCache(email, year)
	if err == nil && !dirty {
		// Convert cache to results
		results := make([]WishlistItem, 0, len(cache))
		for _, item := range cache {
			dbEvents, _ := s.Repo.LoadSimilarEvents(item.EventId, "")
			var entry *events.GenconEvent
			for i := range dbEvents {
				if dbEvents[i].EventId == item.EventId {
					entry = dbEvents[i]
					break
				}
			}
			if entry == nil {
				continue
			}

			results = append(results, WishlistItem{
				Event: StarredEventDetail{
					EventId:          entry.EventId,
					Title:            entry.Title,
					ShortDescription: entry.ShortDescription,
					CategoryCode:     entry.ShortCategory,
					StartTime:        entry.StartTime.Format("2006-01-02T15:04:05Z07:00"),
					EndTime:          entry.EndTime.Format("2006-01-02T15:04:05Z07:00"),
					GenconUrl:        entry.GenconLink(),
					PlannerUrl:       entry.PlannerLink(),
					Tier:             starred.GetTier(entry.EventId),
				},
				Status:    item.Status,
				Reasoning: item.Reasoning,
				Score:     item.Score,
			})
		}
		c.JSON(http.StatusOK, results)
		return
	}


	allStarredEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		log.Printf("error loading starred events: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	constraints, err := s.Repo.GetWishlistConstraints(email)
	if err != nil {
		log.Printf("error loading wishlist constraints: %v\n", err)
		// Fallback to defaults rather than failing
		constraints = []postgres.WishlistConstraint{{DayOfWeek: -1, StartHour: 23, EndHour: 6}}
	}

	// Use the prioritization package
	optimized := prioritization.GeneratePersonalWishlist(starred.StarredEvents, allStarredEvents, constraints)

	// Save to cache
	cacheItems := make([]postgres.WishlistCacheItem, 0, len(optimized))
	for i, item := range optimized {
		cacheItems = append(cacheItems, postgres.WishlistCacheItem{
			EventId:   item.Event.EventId,
			Rank:      i + 1,
			Status:    item.Status,
			Reasoning: item.Reasoning,
			Score:     item.Score,
		})
	}
	s.Repo.SaveWishlistCache(email, year, cacheItems)

	results := make([]WishlistItem, 0, len(optimized))
	for _, item := range optimized {
		results = append(results, WishlistItem{
			Event: StarredEventDetail{
				EventId:          item.Event.EventId,
				Title:            item.Event.Title,
				ShortDescription: item.Event.ShortDescription,
				CategoryCode:     item.Event.ShortCategory,
				StartTime:        item.Event.StartTime.Format("2006-01-02T15:04:05Z07:00"),
				EndTime:          item.Event.EndTime.Format("2006-01-02T15:04:05Z07:00"),
				GenconUrl:        item.Event.GenconLink(),
				PlannerUrl:       item.Event.PlannerLink(),
				Tier:             starred.GetTier(item.Event.EventId),
			},
			Status:    item.Status,
			Reasoning: item.Reasoning,
			Score:     item.Score,
		})
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) GetWishlistConstraints(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	constraints, err := s.Repo.GetWishlistConstraints(email)
	if err != nil {
		log.Printf("error loading wishlist constraints: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	results := make([]WishlistConstraint, 0, len(constraints))
	for _, c := range constraints {
		results = append(results, WishlistConstraint{
			DayOfWeek:          c.DayOfWeek,
			StartHour:          c.StartHour,
			StartMinute:        c.StartMinute,
			EndHour:            c.EndHour,
			EndMinute:          c.EndMinute,
			MinDurationMinutes: c.MinDurationMinutes,
		})
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) UpdateWishlistConstraints(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req []WishlistConstraint
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	dbConstraints := make([]postgres.WishlistConstraint, 0, len(req))
	for _, r := range req {
		// Validate
		if r.DayOfWeek < -1 || r.DayOfWeek > 6 {
			c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid day of week"})
			return
		}
		if r.StartHour < 0 || r.StartHour > 23 || r.EndHour < 0 || r.EndHour > 23 {
			c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Hours must be 0-23"})
			return
		}

		dbConstraints = append(dbConstraints, postgres.WishlistConstraint{
			DayOfWeek:          r.DayOfWeek,
			StartHour:          r.StartHour,
			StartMinute:        r.StartMinute,
			EndHour:            r.EndHour,
			EndMinute:          r.EndMinute,
			MinDurationMinutes: r.MinDurationMinutes,
		})
	}

	err := s.Repo.UpdateWishlistConstraints(email, dbConstraints)
	if err != nil {
		log.Printf("error updating wishlist constraints: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) registerUserRoutes(group *gin.RouterGroup) {
	group.GET("/user", s.GetUser)
	group.GET("/user/events/:email/:year", s.LoadUserEvents)
	group.GET("/user/parties", s.GetParties)
	group.POST("/user/parties", s.CreateParty)
	group.GET("/party/:party_id", s.GetParty)
	group.POST("/party/:party_id/rename", s.RenameParty)
	group.POST("/party/:party_id/transfer", s.TransferLeadership)
	group.POST("/party/:party_id/join", s.JoinParty)
	group.POST("/party/:party_id/leave", s.LeaveParty)
	group.DELETE("/party/:party_id", s.DeleteParty)
	group.POST("/user/rename", s.RenameUser)
	group.GET("/user/starred/:year", s.GetStarredEvents)
	group.GET("/user/starred/list/:year", s.GetStarredIndividualEvents)
	group.GET("/user/starred/calendar/:year", s.GetStarredCalendarEvents)
	group.GET("/user/starred/page/:year", s.GetStarredPageData)
	group.GET("/calendar/metadata/:year", s.GetCalendarMetadata)
	group.POST("/user/star", s.StarEvent)
	group.POST("/user/starred/clear/:year", s.BulkClearStarredEvents)
	group.POST("/user/starred/bulk/:year", s.BulkReplaceStarredEvents)
	group.GET("/user/agenda/:year", s.GetAgenda)
	group.GET("/user/wishlist/:year", s.GetWishlist)
	group.GET("/user/wishlist/constraints", s.GetWishlistConstraints)
	group.POST("/user/wishlist/constraints", s.UpdateWishlistConstraints)
}
