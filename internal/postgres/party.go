package postgres

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// A party is a group of users playing together in a given year.
type Party struct {
	Id          int64
	Name        string
	Year        int64
	LeaderEmail string
	ShortCode   string
	Members     []*User
}

func LoadParties(db *sql.DB, currentUser *User) ([]*Party, error) {
	// Load all partiesById the current user is in
	rows, err := db.Query(
		`
SELECT p.party_id,
       p.name,
       p.year,
       p.leader_email,
       p.short_code
FROM parties p
    JOIN party_members pm ON p.party_id = pm.party_id
WHERE pm.email = $1
`, currentUser.Email)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	partyIds := make([]int64, 0)
	partiesById := make(map[int64]*Party)
	for rows.Next() {
		var p Party
		var leaderEmail sql.NullString
		err = rows.Scan(&p.Id, &p.Name, &p.Year, &leaderEmail, &p.ShortCode)
		if err != nil {
			return nil, err
		}
		p.LeaderEmail = leaderEmail.String
		partiesById[p.Id] = &p
		partyIds = append(partyIds, p.Id)
	}
	rows, err = db.Query(
		`
SELECT pm.party_id,
       pm.email, 
       CASE WHEN length(COALESCE(pm.display_name, '')) > 0 THEN pm.display_name
            WHEN length(COALESCE(u.display_name, '')) > 0 THEN u.display_name
            ELSE split_part(pm.email, '@', 1)
       END,
       COALESCE(pm.gencon_name, u.gencon_name, ''),
       COALESCE(pm.gencon_id, u.gencon_id, ''),
       COALESCE(pm.gencon_email, u.gencon_email, '')
FROM party_members pm JOIN users u ON u.email = pm.email
WHERE pm.party_id = ANY($1)
`, pq.Array(partyIds))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var partyId int64
		var u User
		err = rows.Scan(&partyId, &u.Email, &u.DisplayName, &u.GenconName, &u.GenconId, &u.GenconEmail)
		if err != nil {
			return nil, err
		}
		if p, exists := partiesById[partyId]; exists {
			p.Members = append(p.Members, &u)
		}
	}

	parties := make([]*Party, 0)
	for _, party := range partiesById {
		parties = append(parties, party)
	}

	return parties, nil
}

func NewParty(db *sql.DB, name string, year int64, founderEmail string) (*Party, error) {
	founder, err := LoadOrCreateUser(db, founderEmail)

	if err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	defer func() { CleanupTransaction(err, tx) }()

	shortCode := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	var partyId int64
	err = tx.QueryRow(`
INSERT INTO parties(name, year, leader_email, short_code) VALUES ($1, $2, $3, $4) RETURNING party_id`, name, year, founder.Email, shortCode).Scan(&partyId)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
INSERT INTO party_members (party_id, email, display_name, gencon_name, gencon_id, gencon_email) VALUES ($1, $2, $3, $4, $5, $6)`, partyId, founder.Email, founder.DisplayName, founder.GenconName, founder.GenconId, founder.GenconEmail)
	if err != nil {
		return nil, err
	}

	return &Party{
		Id:          partyId,
		Name:        name,
		Year:        year,
		LeaderEmail: founder.Email,
		ShortCode:   shortCode,
		Members:     []*User{founder},
	}, nil
}

func LoadParty(db *sql.DB, id int64) (*Party, error) {
	var p Party
	var leaderEmail sql.NullString
	err := db.QueryRow(`
SELECT party_id, name, year, leader_email, short_code
FROM parties
WHERE party_id = $1
`, id).Scan(&p.Id, &p.Name, &p.Year, &leaderEmail, &p.ShortCode)
	if err != nil {
		return nil, err
	}
	p.LeaderEmail = leaderEmail.String

	rows, err := db.Query(`
SELECT pm.email, 
       CASE WHEN length(COALESCE(pm.display_name, '')) > 0 THEN pm.display_name
            WHEN length(COALESCE(u.display_name, '')) > 0 THEN u.display_name
            ELSE split_part(pm.email, '@', 1)
       END,
       COALESCE(pm.gencon_name, u.gencon_name, ''),
       COALESCE(pm.gencon_id, u.gencon_id, ''),
       COALESCE(pm.gencon_email, u.gencon_email, '')
FROM party_members pm 
JOIN users u ON u.email = pm.email
WHERE pm.party_id = $1
`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u User
		err = rows.Scan(&u.Email, &u.DisplayName, &u.GenconName, &u.GenconId, &u.GenconEmail)
		if err != nil {
			return nil, err
		}
		p.Members = append(p.Members, &u)
	}

	return &p, nil
}

func LoadPartyByCode(db *sql.DB, code string) (*Party, error) {
	var p Party
	var leaderEmail sql.NullString
	err := db.QueryRow(`
SELECT party_id, name, year, leader_email, short_code
FROM parties
WHERE short_code = $1
`, code).Scan(&p.Id, &p.Name, &p.Year, &leaderEmail, &p.ShortCode)
	if err != nil {
		return nil, err
	}
	p.LeaderEmail = leaderEmail.String

	rows, err := db.Query(`
SELECT pm.email, 
       CASE WHEN length(COALESCE(pm.display_name, '')) > 0 THEN pm.display_name
            WHEN length(COALESCE(u.display_name, '')) > 0 THEN u.display_name
            ELSE split_part(pm.email, '@', 1)
       END,
       COALESCE(pm.gencon_name, u.gencon_name, ''),
       COALESCE(pm.gencon_id, u.gencon_id, ''),
       COALESCE(pm.gencon_email, u.gencon_email, '')
FROM party_members pm 
JOIN users u ON u.email = pm.email
WHERE pm.party_id = $1
`, p.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u User
		err = rows.Scan(&u.Email, &u.DisplayName, &u.GenconName, &u.GenconId, &u.GenconEmail)
		if err != nil {
			return nil, err
		}
		p.Members = append(p.Members, &u)
	}

	return &p, nil
}

func UpdatePartyLeader(db *sql.DB, id int64, newLeaderEmail string) error {
	_, err := db.Exec(`UPDATE parties SET leader_email = $1 WHERE party_id = $2`, newLeaderEmail, id)
	return err
}

func RenameParty(db *sql.DB, id int64, name string) error {
	_, err := db.Exec(`UPDATE parties SET name = $1 WHERE party_id = $2`, name, id)
	return err
}

func DeleteParty(db *sql.DB, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { CleanupTransaction(err, tx) }()

	_, err = tx.Exec(`DELETE FROM party_members WHERE party_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM parties WHERE party_id = $1`, id)
	return err
}

func RemoveMember(db *sql.DB, partyId int64, email string) error {
	_, err := db.Exec(`DELETE FROM party_members WHERE party_id = $1 AND email = $2`, partyId, email)
	return err
}

func JoinParty(db *sql.DB, partyId int64, email string) error {
	// Ensure user exists
	user, err := LoadOrCreateUser(db, email)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
INSERT INTO party_members (party_id, email, display_name, gencon_name, gencon_id, gencon_email)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING
`, partyId, email, user.DisplayName, user.GenconName, user.GenconId, user.GenconEmail)
	return err
}

type MemberInterest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Tier        string `json:"tier"`
}

type SharedInterestGroup struct {
	ClusterId       string           `json:"clusterId"`
	RepEventId      string           `json:"repEventId"`
	AllEventIds     []string         `json:"allEventIds"`
	Title           string           `json:"title"`
	ShortCategory   string           `json:"shortCategory"`
	GameSystem      string           `json:"gameSystem"`
	TotalSessions   int              `json:"totalSessions"`
	TotalTickets    int              `json:"totalTickets"`
	MemberInterests []MemberInterest `json:"memberInterests"`
	GroupScore      int              `json:"groupScore"`
}

func LoadPartySharedInterests(db *sql.DB, partyId int64, year int) ([]*SharedInterestGroup, error) {
	rows, err := db.Query(`
WITH cluster_summary AS (
    SELECT 
        e.cluster_id,
        MIN(e.event_id) as rep_event_id,
        ARRAY_AGG(DISTINCT e.event_id) as all_event_ids,
        e.title,
        e.short_category,
        e.game_system,
        COUNT(DISTINCT e.event_id) as total_sessions,
        SUM(e.tickets_available) as total_tickets
    FROM events e
    JOIN starred_events se ON e.event_id = se.event_id
    JOIN party_members pm ON se.email = pm.email
    WHERE pm.party_id = $1 AND e.year = $2 AND e.active = true AND se.tier != 'not_interested'
    GROUP BY e.cluster_id, e.title, e.short_category, e.game_system
),
member_max_tier AS (
    SELECT 
        e.cluster_id,
        pm.email,
        CASE WHEN length(COALESCE(pm.display_name, '')) > 0 THEN pm.display_name
             WHEN length(COALESCE(u.display_name, '')) > 0 THEN u.display_name
             ELSE split_part(pm.email, '@', 1)
        END as display_name,
        CASE 
            WHEN bool_or(se.tier = 'purchased') THEN 'purchased'
            WHEN bool_or(se.tier = 'must_have') THEN 'must_have'
            WHEN bool_or(se.tier = 'very_interested') THEN 'very_interested'
            WHEN bool_or(se.tier = 'somewhat_interested') THEN 'somewhat_interested'
            WHEN bool_or(se.tier = 'not_interested') THEN 'not_interested'
            ELSE 'not_interested'
        END as max_tier
    FROM party_members pm
    JOIN starred_events se ON pm.email = se.email
    JOIN events e ON se.event_id = e.event_id
    JOIN users u ON pm.email = u.email
    WHERE pm.party_id = $1 AND e.year = $2 AND e.active = true
    GROUP BY e.cluster_id, pm.email, pm.display_name, u.display_name
)
SELECT 
    cs.cluster_id,
    cs.rep_event_id,
    cs.all_event_ids,
    cs.title,
    cs.short_category,
    cs.game_system,
    cs.total_sessions,
    cs.total_tickets,
    JSON_AGG(
        JSONB_BUILD_OBJECT(
            'email', mmt.email,
            'displayName', CASE WHEN length(mmt.display_name) > 0 THEN mmt.display_name ELSE split_part(mmt.email, '@', 1) END,
            'tier', mmt.max_tier
        )
    ) as member_interests
FROM cluster_summary cs
JOIN member_max_tier mmt ON cs.cluster_id = mmt.cluster_id
GROUP BY cs.cluster_id, cs.rep_event_id, cs.all_event_ids, cs.title, cs.short_category, cs.game_system, cs.total_sessions, cs.total_tickets
`, partyId, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*SharedInterestGroup
	for rows.Next() {
		var g SharedInterestGroup
		var memberInterestsJSON []byte
		err := rows.Scan(
			&g.ClusterId,
			&g.RepEventId,
			pq.Array(&g.AllEventIds),
			&g.Title,
			&g.ShortCategory,
			&g.GameSystem,
			&g.TotalSessions,
			&g.TotalTickets,
			&memberInterestsJSON,
		)
		if err != nil {
			return nil, err
		}

		if len(memberInterestsJSON) > 0 {
			if err := json.Unmarshal(memberInterestsJSON, &g.MemberInterests); err != nil {
				return nil, err
			}
		}

		// Calculate GroupScore
		for _, m := range g.MemberInterests {
			switch m.Tier {
			case "must_have":
				g.GroupScore += 100
			case "very_interested":
				g.GroupScore += 50
			case "somewhat_interested":
				g.GroupScore += 10
			}
		}

		groups = append(groups, &g)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].GroupScore > groups[j].GroupScore
	})

	return groups, nil
}

func UpdatePartyMemberInfo(db *sql.DB, partyId int64, email string, displayName string, genconName string, genconId string, genconEmail string) error {
	_, err := db.Exec(`
UPDATE party_members 
SET display_name = $1, gencon_name = $2, gencon_id = $3, gencon_email = $4 
WHERE party_id = $5 AND email = $6
`, displayName, genconName, genconId, genconEmail, partyId, strings.ToLower(email))
	return err
}

