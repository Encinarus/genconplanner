package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/logging"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/Encinarus/genconplanner/internal/prioritization"
	"github.com/Encinarus/genconplanner/internal/pubsub"
	"github.com/gin-gonic/gin"
)

type User struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	GenconName  string `json:"genconName"`
	GenconId    string `json:"genconId"`
	GenconEmail string `json:"genconEmail"`
	IsAdmin     bool   `json:"isAdmin"`
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
	EventId          string                    `json:"eventId"`
	Title            string                    `json:"title"`
	ShortDescription string                    `json:"shortDescription"`
	CategoryCode     string                    `json:"categoryCode"`
	StartTime        string                    `json:"startTime"`
	EndTime          string                    `json:"endTime"`
	GenconUrl        string                    `json:"genconUrl"`
	PlannerUrl       string                    `json:"plannerUrl"`
	Tier             string                    `json:"tier"`
	GroupTier        string                    `json:"groupTier"`
	IsOverride       bool                      `json:"isOverride"`
	Location         string                    `json:"location"`
	RoomName         string                    `json:"roomName"`
	TableNumber      string                    `json:"tableNumber"`
	MapLink          string                    `json:"mapLink,omitempty"`
	PartyMembers     []postgres.MemberInterest `json:"partyMembers"`
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
	Event        StarredEventDetail `json:"event"`
	Status       string             `json:"status"`
	Reasoning    []string           `json:"reasoning"`
	Score        float64            `json:"score"`
	PartyMembers []string           `json:"partyMembers"`
}

type Party struct {
	Id          int64         `json:"id"`
	Name        string        `json:"name"`
	Year        int64         `json:"year"`
	LeaderEmail string        `json:"leaderEmail"`
	ShortCode   string        `json:"shortCode"`
	InviteLink  string        `json:"inviteLink"`
	Members     []PartyMember `json:"members"`
}

func getInviteLink(shortCode string) string {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/party/%s", baseURL, shortCode)
}

type PartyMember struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	GenconName  string `json:"genconName"`
	GenconId    string `json:"genconId"`
	GenconEmail string `json:"genconEmail"`
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

	isAdmin, err := s.Repo.IsAdmin(email)
	if err != nil {
		log.Printf("error checking admin status: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal Server Error"})
		return
	}

	var user User
	user.DisplayName = dbUser.DisplayName
	user.Email = dbUser.Email
	user.GenconName = dbUser.GenconName
	user.GenconId = dbUser.GenconId
	user.GenconEmail = dbUser.GenconEmail
	user.IsAdmin = isAdmin
	c.JSON(http.StatusOK, user)
}

func (s *Server) LoadUserEvents(c *gin.Context) {
	authedEmail := GetUserEmail(c)
	paramEmail := c.Param("email")
	yearParam := c.Param("year")

	if authedEmail == "" || !strings.EqualFold(authedEmail, paramEmail) {
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
	EventId   string `json:"eventId"`
	Add       bool   `json:"add"`
	Related   bool   `json:"related"`
	Tier      string `json:"tier"`
	RemoveAll bool   `json:"removeAll"`
}

func (s *Server) StarEvent(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req StarEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logging.LogCtx(c, "[StarEvent] Error binding JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	logging.LogCtx(c, "[StarEvent] EventId=%s Tier=%s Related=%t Add=%t RemoveAll=%t", req.EventId, req.Tier, req.Related, req.Add, req.RemoveAll)

	var starred *postgres.UserStarredEvents
	var err error
	if req.RemoveAll {
		starred, err = s.Repo.RemoveStarredEventGroup(email, req.EventId)
	} else {
		starred, err = s.Repo.UpdateStarredEventMinimal(email, req.EventId, req.Tier, req.Related, req.Add)
	}

	if err != nil {
		logging.LogCtx(c, "[StarEvent] ERROR updating starred event %s: %v", req.EventId, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	count := 0
	if starred != nil {
		count = len(starred.StarredEvents)
	}
	logging.LogCtx(c, "[StarEvent] SUCCESS eventId=%s returned %d items in payload", req.EventId, count)
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

	partyMembersMap := s.getEventPartyMembers(email, year)

	apiClusters := make([]CalendarEventCluster, 0, len(clusters))
	for _, cluster := range clusters {
		var partyMembers []string
		if pm, found := partyMembersMap[cluster.EventId]; found {
			for _, member := range pm {
				partyMembers = append(partyMembers, member.DisplayName)
			}
		} else {
			partyMembers = make([]string, 0)
		}

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
			Location:         cluster.Location,
			RoomName:         cluster.RoomName,
			TableNumber:      cluster.TableNumber,
			MapLink:          s.MatchLocation(cluster.Location, cluster.RoomName, cluster.TableNumber),
			PartyMembers:     partyMembers,
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

func (s *Server) getEventPartyMembers(email string, year int) map[string][]postgres.MemberInterest {
	eventToPartyMembers := make(map[string][]postgres.MemberInterest)
	user, err := s.Repo.LoadOrCreateUser(email)
	if err != nil {
		return eventToPartyMembers
	}
	dbParties, err := s.Repo.LoadParties(user)
	if err != nil {
		return eventToPartyMembers
	}
	var partyId int64
	for _, p := range dbParties {
		if p.Year == int64(year) {
			partyId = p.Id
			break
		}
	}
	if partyId == 0 {
		return eventToPartyMembers
	}
	partyInterests, err := s.Repo.LoadPartySharedInterests(partyId, year)
	if err != nil {
		return eventToPartyMembers
	}

	for _, group := range partyInterests {
		var members []postgres.MemberInterest
		for _, m := range group.MemberInterests {
			if !strings.EqualFold(m.Email, email) && m.Tier != "not_interested" {
				members = append(members, m)
			}
		}
		if len(members) == 0 {
			members = make([]postgres.MemberInterest, 0)
		}
		for _, eventId := range group.AllEventIds {
			eventToPartyMembers[eventId] = members
		}
	}
	return eventToPartyMembers
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

	partyMembersMap := s.getEventPartyMembers(email, year)
	results := make([]StarredEventDetail, 0)
	starredIds, _ := s.Repo.GetStarredIds(email, year)
	starredMap := make(map[string]postgres.StarredEvent)
	if starredIds != nil {
		for _, s := range starredIds.StarredEvents {
			starredMap[s.EventId] = s
		}
	}

	for _, e := range dbEvents {
		se := starredMap[e.EventId]
		pm, found := partyMembersMap[e.EventId]
		if !found {
			pm = make([]postgres.MemberInterest, 0)
		}
		results = append(results, StarredEventDetail{
			EventId:          e.EventId,
			Title:            e.Title,
			ShortDescription: e.ShortDescription,
			CategoryCode:     e.ShortCategory,
			StartTime:        e.StartTime.Format("2006-01-02T15:04:05Z07:00"),
			EndTime:          e.EndTime.Format("2006-01-02T15:04:05Z07:00"),
			GenconUrl:        e.GenconLink(),
			PlannerUrl:       e.PlannerLink(),
			Tier:             se.Tier,
			GroupTier:        se.GroupTier,
			IsOverride:       se.IsOverride,
			Location:         e.Location,
			RoomName:         e.RoomName,
			TableNumber:      e.TableNumber,
			MapLink:          s.MatchLocation(e.Location, e.RoomName, e.TableNumber),
			PartyMembers:     pm,
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

	// 1. Get starred events (needed for both list and client-side calendar clustering)
	dbEvents, err := s.Repo.LoadStarredEvents(email, year)
	if err != nil {
		logging.LogCtx(c, "[GetStarredPageData] error loading starred events: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// 2. Get Starred IDs for global state sync
	starredIds, err := s.Repo.GetStarredIds(email, year)
	if err != nil {
		logging.LogCtx(c, "[GetStarredPageData] error getting starred ids: %v", err)
	}

	starredCount := 0
	if starredIds != nil {
		starredCount = len(starredIds.StarredEvents)
	}
	logging.LogCtx(c, "[GetStarredPageData] Year=%d: Loaded dbEvents=%d, starredIds=%d", year, len(dbEvents), starredCount)

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

	partyMembersMap := s.getEventPartyMembers(email, year)
	for _, e := range dbEvents {
		tier := ""
		groupTier := ""
		isOverride := false
		if starredIds != nil {
			for _, s := range starredIds.StarredEvents {
				if s.EventId == e.EventId {
					tier = s.Tier
					groupTier = s.GroupTier
					isOverride = s.IsOverride
					break
				}
			}
		}

		pm, found := partyMembersMap[e.EventId]
		if !found {
			pm = make([]postgres.MemberInterest, 0)
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
			GroupTier:        groupTier,
			IsOverride:       isOverride,
			Location:         e.Location,
			RoomName:         e.RoomName,
			TableNumber:      e.TableNumber,
			MapLink:          s.MatchLocation(e.Location, e.RoomName, e.TableNumber),
			PartyMembers:     pm,
		})
	}

	if starredIds != nil {
		for _, s := range starredIds.StarredEvents {
			if s.Tier == "not_interested" {
				continue
			}
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
	Text        string `json:"text"`
	Overwrite   bool   `json:"overwrite"`
	AsGroups    bool   `json:"asGroups"`
	AsPurchased bool   `json:"asPurchased"`
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
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	// 1. Regex to find full event IDs (e.g., BGM26ND323880)
	fullIdRegex := regexp.MustCompile(`[A-Z]{3,4}\d{2}ND\d{5,}`)
	fullMatches := fullIdRegex.FindAllString(req.Text, -1)

	// 2. Regex to find numeric IDs from Gen Con URLs (e.g., gencon.com/events/323880) or bare 5-7 digit numbers
	numericRegex := regexp.MustCompile(`(?i)(?:events/|\b)(\d{5,7})\b`)
	numericSubmatches := numericRegex.FindAllStringSubmatch(req.Text, -1)

	var candidateNumeric []string
	for _, m := range numericSubmatches {
		if len(m) > 1 {
			candidateNumeric = append(candidateNumeric, m[1])
		}
	}

	resolvedMap, err := s.Repo.ResolveNumericEventIds(year, candidateNumeric)
	if err != nil {
		logging.LogCtx(c, "[BulkUpdate] error resolving numeric event IDs: %v", err)
	}

	// 3. Validate that all IDs match the requested year
	yearLastTwo := fmt.Sprintf("%02d", year%100)
	var validIds []string
	seen := make(map[string]bool)

	for _, id := range fullMatches {
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

	if resolvedMap != nil {
		for _, numStr := range candidateNumeric {
			if fullId, found := resolvedMap[numStr]; found {
				if !seen[fullId] {
					validIds = append(validIds, fullId)
					seen[fullId] = true
				}
			}
		}
	}

	if len(validIds) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "No valid event IDs found"})
		return
	}

	logging.LogCtx(c, "[BulkUpdate] Year=%d: Extracted %d IDs from input text. Overwrite=%t AsGroups=%t AsPurchased=%t", year, len(validIds), req.Overwrite, req.AsGroups, req.AsPurchased)
	logging.LogCtx(c, "[BulkUpdate] Valid IDs: %v", validIds)

	err = s.Repo.BulkStarEvents(email, year, validIds, req.Overwrite, req.AsGroups, req.AsPurchased)
	if err != nil {
		logging.LogCtx(c, "[BulkUpdate] ERROR bulk replacing starred events: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	logging.LogCtx(c, "[BulkUpdate] SUCCESS bulk starred %d events for year %d", len(validIds), year)
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

	partyMembersMap := s.getEventPartyMembers(email, year)
	results := make([]StarredEventDetail, 0)
	for _, entry := range dbAgenda {
		pm, found := partyMembersMap[entry.Event.EventId]
		if !found {
			pm = make([]postgres.MemberInterest, 0)
		}
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
			GroupTier:        entry.Tier,
			IsOverride:       true,
			Location:         entry.Event.Location,
			RoomName:         entry.Event.RoomName,
			TableNumber:      entry.Event.TableNumber,
			MapLink:          s.MatchLocation(entry.Event.Location, entry.Event.RoomName, entry.Event.TableNumber),
			PartyMembers:     pm,
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
				GenconName:  m.GenconName,
				GenconId:    m.GenconId,
				GenconEmail: m.GenconEmail,
			})
		}
		apiParties = append(apiParties, Party{
			Id:          p.Id,
			Name:        p.Name,
			Year:        p.Year,
			LeaderEmail: p.LeaderEmail,
			ShortCode:   p.ShortCode,
			InviteLink:  getInviteLink(p.ShortCode),
			Members:     members,
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

	user, err := s.Repo.LoadOrCreateUser(email)
	if err != nil {
		log.Printf("error loading user for create party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	parties, err := s.Repo.LoadParties(user)
	if err == nil {
		for _, p := range parties {
			if p.Year == req.Year {
				c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("You are already a member of a party for %d. You can only be in one party per year.", req.Year)})
				return
			}
		}
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
			GenconName:  m.GenconName,
			GenconId:    m.GenconId,
			GenconEmail: m.GenconEmail,
		})
	}

	apiParty := Party{
		Id:          dbParty.Id,
		Name:        dbParty.Name,
		Year:        dbParty.Year,
		LeaderEmail: dbParty.LeaderEmail,
		ShortCode:   dbParty.ShortCode,
		InviteLink:  getInviteLink(dbParty.ShortCode),
		Members:     members,
	}

	c.JSON(http.StatusOK, apiParty)
}

func (s *Server) getPartyFromParam(idParam string, email string, allowShortCode bool) (*postgres.Party, error) {
	id, parseErr := strconv.ParseInt(idParam, 10, 64)
	if parseErr == nil {
		if id >= 2000 && id <= 2100 {
			// It's a year. Load user's party for this year.
			user, err := s.Repo.LoadOrCreateUser(email)
			if err != nil {
				return nil, err
			}
			parties, err := s.Repo.LoadParties(user)
			if err != nil {
				return nil, err
			}
			for _, p := range parties {
				if p.Year == id {
					return p, nil
				}
			}
			return nil, fmt.Errorf("no party found for year %d", id)
		}
		// Fallback: it's a numeric party ID
		party, err := s.Repo.LoadParty(id)
		if err != nil {
			return nil, err
		}
		if email != "" {
			for _, m := range party.Members {
				if strings.EqualFold(m.Email, email) {
					return party, nil
				}
			}
		}
		return nil, fmt.Errorf("party not found")
	}
	if !allowShortCode {
		return nil, fmt.Errorf("short code lookup not permitted for this operation")
	}
	// It's a short code
	return s.Repo.LoadPartyByCode(idParam)
}

func (s *Server) GetParty(c *gin.Context) {
	idParam := c.Param("party_id")
	email := GetUserEmail(c)
	dbParty, err := s.getPartyFromParam(idParam, email, true)

	if err != nil {
		log.Printf("error loading party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	isMember := false
	if email != "" {
		for _, m := range dbParty.Members {
			if m.Email == email {
				isMember = true
				break
			}
		}
	}

	_, parseErr := strconv.ParseInt(idParam, 10, 64)
	if parseErr == nil {
		// If loaded by numeric ID or year, require membership
		if !isMember {
			c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
			return
		}
	}

	members := make([]PartyMember, 0, len(dbParty.Members))
	for _, m := range dbParty.Members {
		if isMember {
			members = append(members, PartyMember{
				DisplayName: m.DisplayName,
				Email:       m.Email,
				GenconName:  m.GenconName,
				GenconId:    m.GenconId,
				GenconEmail: m.GenconEmail,
			})
		} else {
			// Redact sensitive PII for non-members previewing the party via shortCode.
			// Preserve the leader's email so the UI can identify the leader badge, otherwise redact.
			memberEmail := ""
			if m.Email == dbParty.LeaderEmail {
				memberEmail = m.Email
			}
			members = append(members, PartyMember{
				DisplayName: m.DisplayName,
				Email:       memberEmail,
				GenconName:  "",
				GenconId:    "",
				GenconEmail: "",
			})
		}
	}

	apiParty := Party{
		Id:          dbParty.Id,
		Name:        dbParty.Name,
		Year:        dbParty.Year,
		LeaderEmail: dbParty.LeaderEmail,
		ShortCode:   dbParty.ShortCode,
		InviteLink:  getInviteLink(dbParty.ShortCode),
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

	dbParty, err := s.getPartyFromParam(c.Param("party_id"), email, false)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	var req RenamePartyRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	if dbParty.LeaderEmail != email {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Only the Party Leader can rename the party"})
		return
	}

	err = s.Repo.RenameParty(dbParty.Id, req.Name)
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

	dbParty, err := s.getPartyFromParam(c.Param("party_id"), email, false)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	var req TransferLeadershipRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	if dbParty.LeaderEmail != email {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Only the Party Leader can transfer leadership"})
		return
	}

	// Verify new leader is a member
	isMember := false
	for _, m := range dbParty.Members {
		if m.Email == req.NewLeaderEmail {
			isMember = true
			break
		}
	}

	if !isMember {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "New leader must be a member of the party"})
		return
	}

	err = s.Repo.UpdatePartyLeader(dbParty.Id, req.NewLeaderEmail)
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

	idParam := c.Param("party_id")
	if _, err := strconv.ParseInt(idParam, 10, 64); err == nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Join requests must use the party invite code"})
		return
	}

	dbParty, err := s.Repo.LoadPartyByCode(idParam)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	user, err := s.Repo.LoadOrCreateUser(email)
	if err != nil {
		log.Printf("error loading user for join party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	parties, err := s.Repo.LoadParties(user)
	if err == nil {
		for _, p := range parties {
			if p.Year == dbParty.Year {
				c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("You are already a member of a party for %d. You can only be in one party per year.", dbParty.Year)})
				return
			}
		}
	}

	err = s.Repo.JoinParty(dbParty.Id, email)
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

	dbParty, err := s.getPartyFromParam(c.Param("party_id"), email, false)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	isMember := false
	for _, m := range dbParty.Members {
		if strings.EqualFold(m.Email, email) {
			isMember = true
			break
		}
	}
	if !isMember {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	if dbParty.LeaderEmail == email && len(dbParty.Members) > 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Party Leader cannot leave unless they transfer leadership or are the last member"})
		return
	}

	err = s.Repo.RemoveMember(dbParty.Id, email)
	if err != nil {
		log.Printf("error leaving party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	// If last member leaves, delete the party
	if len(dbParty.Members) == 1 {
		err = s.Repo.DeleteParty(dbParty.Id)
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

	dbParty, err := s.getPartyFromParam(c.Param("party_id"), email, false)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	if dbParty.LeaderEmail != email {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Only the Party Leader can delete the party"})
		return
	}

	if len(dbParty.Members) > 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Party can only be deleted if there is only one member remaining"})
		return
	}

	err = s.Repo.DeleteParty(dbParty.Id)
	if err != nil {
		log.Printf("error deleting party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type RenameUserRequest struct {
	DisplayName string `json:"displayName"`
	GenconName  string `json:"genconName"`
	GenconId    string `json:"genconId"`
	GenconEmail string `json:"genconEmail"`
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

	err := s.Repo.UpdateUserGenconInfo(email, req.DisplayName, req.GenconName, req.GenconId, req.GenconEmail)
	if err != nil {
		log.Printf("error renaming user: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type UpdatePartyMemberRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	GenconName  string `json:"genconName"`
	GenconId    string `json:"genconId"`
	GenconEmail string `json:"genconEmail"`
}

func (s *Server) UpdatePartyMemberInfo(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req UpdatePartyMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request"})
		return
	}

	if req.DisplayName == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Display name cannot be empty"})
		return
	}

	idParam := c.Param("party_id")
	dbParty, err := s.getPartyFromParam(idParam, email, false)
	if err != nil {
		log.Printf("error loading party: %v\n", err)
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	// Verify target user is a member of the party
	isTargetMember := false
	for _, m := range dbParty.Members {
		if strings.EqualFold(m.Email, req.Email) {
			isTargetMember = true
			break
		}
	}
	if !isTargetMember {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Target user is not a member of this party"})
		return
	}

	// Enforce authorization: only party leader or the member themselves
	if !strings.EqualFold(dbParty.LeaderEmail, email) && !strings.EqualFold(email, req.Email) {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Only the party leader or the member themselves can update member info"})
		return
	}

	err = s.Repo.UpdatePartyMemberInfo(dbParty.Id, req.Email, req.DisplayName, req.GenconName, req.GenconId, req.GenconEmail)
	if err != nil {
		log.Printf("error updating party member info: %v\n", err)
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

	var results []WishlistItem

	cache, dirty, updatedAt, err := s.Repo.GetWishlistCache(email, year)
	if err == nil && !dirty {
		// Convert cache to results
		results = make([]WishlistItem, 0, len(cache))
		partyMembersMap := s.getEventPartyMembers(email, year)
		for _, item := range cache {
			dbEvents, _ := s.Repo.LoadSimilarEvents(c.Request.Context(), item.EventId, "")
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

			pm, found := partyMembersMap[entry.EventId]
			if !found {
				pm = make([]postgres.MemberInterest, 0)
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
					Location:         entry.Location,
					RoomName:         entry.RoomName,
					TableNumber:      entry.TableNumber,
					MapLink:          s.MatchLocation(entry.Location, entry.RoomName, entry.TableNumber),
					PartyMembers:     pm,
				},
				Status:    item.Status,
				Reasoning: item.Reasoning,
				Score:     item.Score,
			})
		}
	} else {
		allStarredEvents, loadErr := s.Repo.LoadStarredEvents(email, year)
		if loadErr != nil {
			log.Printf("error loading starred events: %v\n", loadErr)
			c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
			return
		}

		constraints, constErr := s.Repo.GetWishlistConstraints(email)
		if constErr != nil {
			log.Printf("error loading wishlist constraints: %v\n", constErr)
			// Fallback to defaults rather than failing
			constraints = []postgres.WishlistConstraint{{DayOfWeek: -1, StartHour: 23, EndHour: 6}}
		}

		var partyId int64
		user, userErr := s.Repo.LoadOrCreateUser(email)
		if userErr == nil {
			dbParties, partiesErr := s.Repo.LoadParties(user)
			if partiesErr == nil {
				for _, p := range dbParties {
					if p.Year == int64(year) {
						partyId = p.Id
						break
					}
				}
			}
		}

		var partyPurchases map[string]int
		if partyId != 0 {
			partyPurchases, _ = s.Repo.LoadPartyMemberPurchases(partyId, year)
		}

		// Use the prioritization package
		optimized := prioritization.GeneratePersonalWishlist(starred.StarredEvents, allStarredEvents, constraints, partyPurchases)

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
		if saveErr := s.Repo.SaveWishlistCache(email, year, cacheItems, updatedAt); saveErr != nil {
			log.Printf("error saving wishlist cache: %v\n", saveErr)
		}

		partyMembersMap := s.getEventPartyMembers(email, year)
		results = make([]WishlistItem, 0, len(optimized))
		for _, item := range optimized {
			pm, found := partyMembersMap[item.Event.EventId]
			if !found {
				pm = make([]postgres.MemberInterest, 0)
			}
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
					Location:         item.Event.Location,
					RoomName:         item.Event.RoomName,
					TableNumber:      item.Event.TableNumber,
					MapLink:          s.MatchLocation(item.Event.Location, item.Event.RoomName, item.Event.TableNumber),
					PartyMembers:     pm,
				},
				Status:    item.Status,
				Reasoning: item.Reasoning,
				Score:     item.Score,
			})
		}
	}

	user, err := s.Repo.LoadOrCreateUser(email)
	if err == nil {
		dbParties, err := s.Repo.LoadParties(user)
		if err == nil {
			var partyId int64
			for _, p := range dbParties {
				if p.Year == int64(year) {
					partyId = p.Id
					break
				}
			}
			if partyId != 0 {
				partyInterests, err := s.Repo.LoadPartySharedInterests(partyId, year)
				if err == nil {
					eventToPartyMembers := make(map[string][]string)
					for _, group := range partyInterests {
						var memberNames []string
						for _, m := range group.MemberInterests {
							if !strings.EqualFold(m.Email, email) && m.Tier != "not_interested" {
								memberNames = append(memberNames, m.DisplayName)
							}
						}
						if len(memberNames) == 0 {
							memberNames = make([]string, 0)
						}
						for _, eventId := range group.AllEventIds {
							eventToPartyMembers[eventId] = memberNames
						}
					}

					for i := range results {
						if members, found := eventToPartyMembers[results[i].Event.EventId]; found {
							results[i].PartyMembers = members
						} else {
							results[i].PartyMembers = make([]string, 0)
						}
					}
				}
			}
		}
	}

	// Ensure PartyMembers is initialized to empty slice for all items if party lookup didn't happen
	for i := range results {
		if results[i].PartyMembers == nil {
			results[i].PartyMembers = make([]string, 0)
		}
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

func (s *Server) GetPartyInterests(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	idParam := c.Param("party_id")
	dbParty, err := s.getPartyFromParam(idParam, email, true)

	if err != nil || dbParty == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	// Verify membership
	isMember := false
	for _, m := range dbParty.Members {
		if m.Email == email {
			isMember = true
			break
		}
	}
	if !isMember {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	yearParam := c.Query("year")
	year, err := strconv.Atoi(yearParam)
	if err != nil {
		year = time.Now().Year()
	}

	groups, err := s.Repo.LoadPartySharedInterests(dbParty.Id, year)
	if err != nil {
		log.Printf("error loading party shared interests: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (s *Server) PartyStream(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	idParam := c.Param("party_id")
	dbParty, err := s.getPartyFromParam(idParam, email, true)

	if err != nil || dbParty == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	// Verify membership
	isMember := false
	for _, m := range dbParty.Members {
		if m.Email == email {
			isMember = true
			break
		}
	}
	if !isMember {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: "Party not found"})
		return
	}

	sub := pubsub.Subscribe(dbParty.Id)
	defer sub.Unsubscribe()

	// Instantiate a single ticker outside the callback to prevent timer accumulation
	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	c.Stream(func(w io.Writer) bool {
		// Stop the stream if the context is aborted (e.g. write error)
		if c.IsAborted() {
			return false
		}
		select {
		case ev := <-sub.C:
			c.SSEvent("interest_update", ev)
			return !c.IsAborted()
		case <-pingTicker.C:
			c.SSEvent("ping", map[string]string{"status": "heartbeat"})
			return !c.IsAborted()
		case <-c.Request.Context().Done():
			return false
		}
	})
}

func (s *Server) registerUserRoutes(group *gin.RouterGroup) {
	group.GET("/user", s.GetUser)
	group.GET("/user/events/:email/:year", s.LoadUserEvents)
	group.GET("/user/parties", s.GetParties)
	group.POST("/user/parties", s.CreateParty)
	group.GET("/party/:party_id", s.GetParty)
	group.GET("/party/:party_id/interests", s.GetPartyInterests)
	group.GET("/party/:party_id/stream", s.PartyStream)
	group.POST("/party/:party_id/rename", s.RenameParty)
	group.POST("/party/:party_id/transfer", s.TransferLeadership)
	group.POST("/party/:party_id/join", s.JoinParty)
	group.POST("/party/:party_id/leave", s.LeaveParty)
	group.DELETE("/party/:party_id", s.DeleteParty)
	group.POST("/party/:party_id/member/update", s.UpdatePartyMemberInfo)
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
	s.registerTicketRoutes(group)
}
