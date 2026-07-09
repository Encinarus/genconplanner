package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/lib/pq"
)

type User struct {
	Email       string
	DisplayName string
	GenconName  string
	GenconId    string
	GenconEmail string
}

type StarredEvent struct {
	EventId    string
	Level      string // "group" or "event"
	Tier       string // "must_have", "very_interested", "somewhat_interested", "not_interested"
	GroupTier  string
	IsOverride bool
}

type UserStarredEvents struct {
	Email         string
	StarredEvents []StarredEvent
}

type WishlistConstraint struct {
	DayOfWeek          int // -1 for Every Day, 0-6 for Sun-Sat
	StartHour          int
	StartMinute        int
	EndHour            int
	EndMinute          int
	MinDurationMinutes int // 0 means hard block, > 0 means flexible block
}

func (u *UserStarredEvents) GetTier(eventId string) string {
	for _, s := range u.StarredEvents {
		if s.EventId == eventId {
			return s.Tier
		}
	}
	return ""
}

func UpdateDisplayName(db *sql.DB, email string, displayName string) error {
	email = strings.ToLower(email)
	_, err := db.Exec("UPDATE users SET display_name = $1 WHERE email = $2", displayName, email)
	return err
}

func UpdateUserGenconInfo(db *sql.DB, email string, displayName string, genconName string, genconId string, genconEmail string) error {
	email = strings.ToLower(email)
	_, err := db.Exec("UPDATE users SET display_name = $1, gencon_name = $2, gencon_id = $3, gencon_email = $4 WHERE email = $5", displayName, genconName, genconId, genconEmail, email)
	return err
}

func LoadStarredEventClusters(db *sql.DB, userEmail string, year int, starredEvents []*events.GenconEvent) ([]*CalendarEventCluster, error) {
	rows, err := db.Query(`
SELECT 
    CASE e.day_of_week 
		WHEN 3 THEN 'wed'
		WHEN 4 THEN 'thu'
		WHEN 5 THEN 'fri'
		WHEN 6 THEN 'sat'
		WHEN 0 THEN 'sun'
	END AS day_of_week,
    ARRAY_AGG(e.event_id) event_ids,
    ARRAY_AGG(COALESCE(override.tier, grp.tier)) tiers
FROM events e 
JOIN starred_events grp ON grp.email = $1 AND grp.level = 'group'
JOIN events e1 ON grp.event_id = e1.event_id 
    AND e1.year = e.year 
    AND e1.short_category = e.short_category 
    AND e1.title = e.title 
    AND e1.short_description = e.short_description
LEFT JOIN starred_events override ON override.email = $1 AND override.event_id = e.event_id AND override.level = 'event'
WHERE e.year = $2
  AND e.active
  AND COALESCE(override.tier, grp.tier) != 'not_interested'
GROUP BY e.cluster_key, e.day_of_week
`, userEmail, year)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	eventsById := make(map[string]*events.GenconEvent)
	for _, e := range starredEvents {
		eventsById[e.EventId] = e
	}

	groupedEvents := make([]*CalendarEventCluster, 0)
	for rows.Next() {
		eventIds := make([]string, 0)
		tiers := make([]string, 0)
		var dayOfWeek string
		err = rows.Scan(&dayOfWeek, pq.Array(&eventIds), pq.Array(&tiers))
		if err != nil {
			return nil, err
		}

		eventTiers := make(map[string]string)
		for i, id := range eventIds {
			eventTiers[id] = tiers[i]
		}

		dayGroupEvents := make([]*events.GenconEvent, 0, len(eventIds))
		for _, id := range eventIds {
			// Guard against events being starred between the load and
			// here. Should be _super_ rare, handle anyway.
			if e, present := eventsById[id]; present {
				dayGroupEvents = append(dayGroupEvents, e)
			} else {
				log.Printf("Can't find event %v in events", id)
			}
		}
		// We sort the events by start time so we can reference
		// the earliest one in each cluster
		sort.Slice(dayGroupEvents, func(i, j int) bool {
			return dayGroupEvents[i].StartTime.Before(dayGroupEvents[j].StartTime)
		})

		cluster := newClusterForEvent(dayGroupEvents[0])
		clusterTier := eventTiers[cluster.EventId]

		for _, event := range dayGroupEvents[1:] {
			eventTier := eventTiers[event.EventId]
			if event.StartTime.After(cluster.EndTime) || eventTier == "purchased" || clusterTier == "purchased" {
				if cluster.SimilarCount > 1 {
					cluster.Title = fmt.Sprintf("%s\n\n(%d similar)", cluster.Title, cluster.SimilarCount)
				}
				groupedEvents = append(groupedEvents, cluster)
				cluster = newClusterForEvent(event)
				clusterTier = eventTier
			} else {
				if event.EndTime.After(cluster.EndTime) {
					cluster.EndTime = event.EndTime
				}
				cluster.SimilarCount++
			}
		}

		if cluster.SimilarCount > 1 {
			cluster.Title = fmt.Sprintf("%s\n\n(%d similar)", cluster.Title, cluster.SimilarCount)
		}
		groupedEvents = append(groupedEvents, cluster)
	}

	log.Printf("Returning %v groups", len(groupedEvents))
	return groupedEvents, nil
}

func LoadStarredEventGroups(db *sql.DB, userEmail string, year int) ([]*EventGroup, error) {
	rows, err := db.Query(`
SELECT
	MIN(e2.event_id) AS anchor_event,
	e2.title, 
	e2.short_description AS short_description,
	e2.short_category AS short_category,
	e2.game_system AS game_system,
	e2.org_group AS org_group,
	MAX(o.id) AS org_id,
	COUNT(*) AS num_events,
	SUM(e2.tickets_available) AS tickets_available,
	sum(CASE WHEN e2.day_of_week = 3 THEN e2.tickets_available ELSE 0 END) as wednesday_tickets,
	sum(CASE WHEN e2.day_of_week = 4 THEN e2.tickets_available ELSE 0 END) as thursday_tickets,
	sum(CASE WHEN e2.day_of_week = 5 THEN e2.tickets_available ELSE 0 END) as friday_tickets,
	sum(CASE WHEN e2.day_of_week = 6 THEN e2.tickets_available ELSE 0 END) as saturday_tickets,
	sum(CASE WHEN e2.day_of_week = 0 THEN e2.tickets_available ELSE 0 END) as sunday_tickets,
	0 as title_rank,
	0 as search_rank
FROM events e2
LEFT JOIN (
    SELECT lower(alias) as lower_alias, MAX(id) as id
    FROM orgs
    GROUP BY lower(alias)
) o ON o.lower_alias = lower(e2.org_group)
WHERE e2.active AND e2.year = $2
  AND EXISTS (
      SELECT 1 FROM starred_events se
      JOIN events e1 ON se.event_id = e1.event_id
      WHERE se.email = $1
        AND e1.year = e2.year AND e1.short_category = e2.short_category AND e1.title = e2.title AND e1.short_description = e2.short_description
  )
GROUP BY
  e2.year, e2.short_category, e2.title, e2.short_description, e2.game_system, e2.org_group
ORDER BY e2.title`, userEmail, year)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]*EventGroup, 0)
	for rows.Next() {
		group, err := rowToGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func LoadStarredEvents(db *sql.DB, userEmail string, year int) ([]*events.GenconEvent, error) {
	fields := "e2." + strings.Join(eventFields(), ", e2.")
	rows, err := db.Query(fmt.Sprintf(`
SELECT %s, true, o.id
FROM events e2
LEFT JOIN (
    SELECT lower(alias) as lower_alias, MAX(id) as id
    FROM orgs
    GROUP BY lower(alias)
) o ON o.lower_alias = lower(e2.org_group)
WHERE e2.active AND e2.year = $2
  AND EXISTS (
      SELECT 1 FROM starred_events se
      JOIN events e1 ON se.event_id = e1.event_id
      WHERE se.email = $1
        AND e1.year = e2.year AND e1.short_category = e2.short_category AND e1.title = e2.title AND e1.short_description = e2.short_description
  )
ORDER BY e2.start_time, e2.event_id`, fields), userEmail, year)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loadedEvents := make([]*events.GenconEvent, 0)
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		loadedEvents = append(loadedEvents, event)
	}
	return loadedEvents, nil
}

func UpdateStarredEvent(db *sql.DB, email string, eventId string, tier string, starGroup bool, add bool) (*UserStarredEvents, error) {
	return UpdateStarredEventInternal(db, email, eventId, tier, starGroup, add, true)
}

func UpdateStarredEventMinimal(db *sql.DB, email string, eventId string, tier string, starGroup bool, add bool) (*UserStarredEvents, error) {
	return UpdateStarredEventInternal(db, email, eventId, tier, starGroup, add, false)
}

func RemoveStarredEventGroup(db *sql.DB, email string, eventId string) (*UserStarredEvents, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		DELETE FROM starred_events WHERE email = $1 AND event_id IN (
			SELECT e2.event_id FROM events e1 JOIN events e2 ON e1.cluster_id = e2.cluster_id WHERE e1.event_id = $2
		)`, email, eventId)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return GetStarredIds(db, email, events.YearFromEvent(eventId))
}

func UpdateStarredEventInternal(db *sql.DB, email string, eventId string, tier string, starGroup bool, add bool, fullResponse bool) (*UserStarredEvents, error) {
	if tier == "" {
		tier = "very_interested"
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if starGroup {
		if add {
			// Update the group default tier
			// 1. Check if there's already a level = 'group' row for this group
			res, execErr := tx.Exec(`
				UPDATE starred_events SET tier = $3 
				WHERE email = $1 AND level = 'group' AND event_id IN (
					SELECT e2.event_id FROM events e1 JOIN events e2 ON e1.cluster_id = e2.cluster_id WHERE e1.event_id = $2
				)`, email, eventId, tier)
			if execErr != nil {
				return nil, execErr
			}
			rowsAffected, _ := res.RowsAffected()
			if rowsAffected == 0 {
				// No group default row existed yet, insert one
				_, err = tx.Exec(`
					INSERT INTO starred_events (email, event_id, level, tier) VALUES ($1, $2, 'group', $3)
					ON CONFLICT (event_id, email, level) DO UPDATE SET tier = $3
				`, email, eventId, tier)
				if err != nil {
					return nil, err
				}
			}
		} else {
			// Unstar group (user clicked the same value for the group default).
			// Check if there are any specific instances with an override (level = 'event').
			var overrideEventId string
			var overrideTier string
			scanErr := tx.QueryRow(`
				SELECT se.event_id, se.tier 
				FROM starred_events se
				JOIN events e1 ON se.event_id = e1.event_id
				JOIN events e2 ON e1.cluster_id = e2.cluster_id
				WHERE e2.event_id = $2 AND se.email = $1 AND se.level = 'event'
				ORDER BY e1.event_id LIMIT 1
			`, email, eventId).Scan(&overrideEventId, &overrideTier)

			if scanErr == sql.ErrNoRows {
				// If no specific instances have an override, this should remove it entirely from the schedule.
				_, err = tx.Exec(`
					DELETE FROM starred_events WHERE email = $1 AND event_id IN (
						SELECT e2.event_id FROM events e1 JOIN events e2 ON e1.cluster_id = e2.cluster_id WHERE e1.event_id = $2
					)`, email, eventId)
				if err != nil {
					return nil, err
				}
			} else if scanErr != nil {
				return nil, scanErr
			} else {
				// If there are specific instances with an override, the default should be updated to match the instance with the override, and the override on that specific instance should be removed.
				// 1. Delete the old group default row(s).
				_, err = tx.Exec(`
					DELETE FROM starred_events WHERE email = $1 AND level = 'group' AND event_id IN (
						SELECT e2.event_id FROM events e1 JOIN events e2 ON e1.cluster_id = e2.cluster_id WHERE e1.event_id = $2
					)`, email, eventId)
				if err != nil {
					return nil, err
				}
				// 2. Update the override instance to become the new group default.
				_, err = tx.Exec(`
					UPDATE starred_events SET level = 'group' WHERE email = $1 AND event_id = $2
				`, email, overrideEventId)
				if err != nil {
					return nil, err
				}
			}
		}
	} else {
		if add {
			// Upsert explicit override
			_, err = tx.Exec(`
				INSERT INTO starred_events (email, event_id, level, tier) VALUES ($1, $2, 'event', $3)
				ON CONFLICT (event_id, email, level) DO UPDATE SET tier = $3
			`, email, eventId, tier)

			if err != nil {
				return nil, err
			}
		} else {
			// Remove explicit override
			_, err = tx.Exec("DELETE FROM starred_events WHERE email = $1 AND event_id = $2 AND level = 'event'", email, eventId)
			if err != nil {
				return nil, err
			}
		}
	}

	err = NormalizeUserStarredEvents(tx, email, 0)
	if err != nil {
		return nil, err
	}

	// Mark wishlist as dirty
	_, err = tx.Exec("UPDATE users SET wishlist_dirty = TRUE, wishlist_updated_at = NOW() WHERE email = $1", email)
	if err != nil {
		return nil, err
	}

	if !fullResponse {
		err = tx.Commit()
		if err != nil {
			return nil, err
		}
		level := "event"
		if starGroup {
			level = "group"
		}
		return &UserStarredEvents{
			Email: email,
			StarredEvents: []StarredEvent{{
				EventId:    eventId,
				Level:      level,
				Tier:       tier,
				GroupTier:  tier,
				IsOverride: !starGroup,
			}},
		}, nil
	}

	starredEvents, err := fetchStarredInternal(tx, email, 0)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return starredEvents, nil
}

func ClearStarredEvents(db *sql.DB, email string, year int) error {
	_, err := db.Exec(`
DELETE FROM starred_events se
USING events e
WHERE se.event_id = e.event_id
  AND se.email = $1
  AND e.year = $2
`, email, year)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE users SET wishlist_dirty = TRUE, wishlist_updated_at = NOW() WHERE email = $1", email)
	return err
}

func BulkStarEvents(db *sql.DB, email string, year int, eventIds []string, overwrite bool, asGroups bool, asPurchased bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if overwrite {
		// 1. Clear existing for the year
		_, err = tx.Exec(`
DELETE FROM starred_events se
USING events e
WHERE se.event_id = e.event_id
  AND se.email = $1
  AND e.year = $2
`, email, year)
		if err != nil {
			return err
		}
	}

	// 2. Insert new ones
	if len(eventIds) > 0 {
		level := "event"
		tier := "very_interested"
		if asPurchased {
			level = "event"
			tier = "purchased"
		} else if asGroups {
			level = "group"
		}

		for _, id := range eventIds {
			_, err = tx.Exec(`
INSERT INTO starred_events (email, event_id, level, tier)
VALUES ($1, $2, $3, $4)
ON CONFLICT (event_id, email, level) DO UPDATE SET tier = EXCLUDED.tier
`, email, id, level, tier)
			if err != nil {
				return err
			}
		}
	}

	err = NormalizeUserStarredEvents(tx, email, year)
	if err != nil {
		return err
	}

	// Mark wishlist as dirty
	_, err = tx.Exec("UPDATE users SET wishlist_dirty = TRUE, wishlist_updated_at = NOW() WHERE email = $1", email)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func GetStarredIds(db *sql.DB, email string, year int) (*UserStarredEvents, error) {
	return fetchStarredInternal(db, email, year)
}

func GetAllStarredIds(db *sql.DB, email string) (*UserStarredEvents, error) {
	return fetchStarredInternal(db, email, 0)
}

type queryable interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func NormalizeUserStarredEvents(q queryable, email string, year int) error {
	yearFilter := ""
	args := []interface{}{email}
	if year > 0 {
		yearFilter = " AND e1.year = $2"
		args = append(args, year)
	}

	query := fmt.Sprintf(`
SELECT e.year, e.short_category, e.title, e.short_description,
       ARRAY_AGG(e.event_id ORDER BY e.event_id) as all_event_ids,
       ARRAY_AGG(se.level ORDER BY e.event_id) as starred_levels,
       ARRAY_AGG(se.tier ORDER BY e.event_id) as starred_tiers
FROM events e
JOIN (
    SELECT DISTINCT e1.year, e1.short_category, e1.title, e1.short_description
    FROM starred_events se1
    JOIN events e1 ON se1.event_id = e1.event_id
    WHERE se1.email = $1 %s
) active_groups ON e.year = active_groups.year 
    AND e.short_category = active_groups.short_category 
    AND e.title = active_groups.title 
    AND e.short_description = active_groups.short_description
LEFT JOIN starred_events se ON se.event_id = e.event_id AND se.email = $1
GROUP BY e.year, e.short_category, e.title, e.short_description
`, yearFilter)

	rows, err := q.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	type groupData struct {
		allEventIds   []string
		starredLevels []sql.NullString
		starredTiers  []sql.NullString
	}
	var groups []groupData

	for rows.Next() {
		var g groupData
		var y int
		var cat, title, desc string
		if err := rows.Scan(&y, &cat, &title, &desc, pq.Array(&g.allEventIds), pq.Array(&g.starredLevels), pq.Array(&g.starredTiers)); err != nil {
			return err
		}
		groups = append(groups, g)
	}

	tierPriority := func(tier string) int {
		switch tier {
		case "purchased":
			return 5
		case "must_have":
			return 4
		case "very_interested":
			return 3
		case "somewhat_interested":
			return 2
		case "not_interested":
			return 1
		default:
			return 0
		}
	}

	for _, g := range groups {
		var groupLevelIndices []int
		var starredIndices []int
		maxTier := ""
		maxTierPriority := -1
		hasNonNotInterested := false

		for i := range g.allEventIds {
			if g.starredLevels[i].Valid {
				starredIndices = append(starredIndices, i)
				if g.starredLevels[i].String == "group" {
					groupLevelIndices = append(groupLevelIndices, i)
				}
				t := g.starredTiers[i].String
				if t != "not_interested" {
					hasNonNotInterested = true
				}
				if t == "purchased" {
					continue // Purchased state is only valid on an override, ignored when picking group default.
				}
				p := tierPriority(t)
				if p > maxTierPriority {
					maxTierPriority = p
					maxTier = t
				}
			}
		}

		if len(groupLevelIndices) == 0 && !hasNonNotInterested {
			// Clean up ghost rows so the group is correctly considered unset!
			for _, idx := range starredIndices {
				_, err := q.Exec("DELETE FROM starred_events WHERE email = $1 AND event_id = $2", email, g.allEventIds[idx])
				if err != nil {
					return err
				}
			}
			continue
		}

		if len(groupLevelIndices) > 1 {
			// Property 1: two or more instances are considered to be representing the group.
			// The one with higher rating should be considered the group default, lower priority demoted to instance override.
			bestIdx := groupLevelIndices[0]
			bestPrio := tierPriority(g.starredTiers[bestIdx].String)

			for _, idx := range groupLevelIndices[1:] {
				p := tierPriority(g.starredTiers[idx].String)
				if p > bestPrio {
					bestPrio = p
					bestIdx = idx
				}
			}

			// Demote all others
			for _, idx := range groupLevelIndices {
				if idx != bestIdx {
					// Check if an 'event' row already exists. If so, just delete the 'group' row. Otherwise update 'group' to 'event'.
					_, err := q.Exec(`
						INSERT INTO starred_events (email, event_id, level, tier)
						SELECT email, event_id, 'event', tier FROM starred_events WHERE email = $1 AND event_id = $2 AND level = 'group'
						ON CONFLICT (event_id, email, level) DO NOTHING;
						DELETE FROM starred_events WHERE email = $1 AND event_id = $2 AND level = 'group';
					`, email, g.allEventIds[idx])
					if err != nil {
						return err
					}
				}
			}
		} else if len(groupLevelIndices) == 0 {
			// Property 2: no event group currently set up, but there are active starred event instances.
			// Add a default star rating for the group equal to the highest rating among the events that are starred.
			// By default it should be the one with the event id that sorts the lowest among unstarred events in the group.
			if maxTier == "" {
				maxTier = "very_interested"
			}
			if len(g.allEventIds) > 0 {
				defaultEventId := g.allEventIds[0]
				for i, id := range g.allEventIds {
					if !g.starredLevels[i].Valid {
						defaultEventId = id
						break
					}
				}
				_, err := q.Exec("INSERT INTO starred_events (email, event_id, level, tier) VALUES ($1, $2, 'group', $3) ON CONFLICT (event_id, email, level) DO UPDATE SET tier = $3", email, defaultEventId, maxTier)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func fetchStarredInternal(q queryable, email string, year int) (*UserStarredEvents, error) {
	err := NormalizeUserStarredEvents(q, email, year)
	if err != nil {
		return nil, err
	}

	starredEvents := UserStarredEvents{
		Email: email,
	}

	yearFilter := ""
	args := []interface{}{email}
	if year > 0 {
		yearFilter = " AND e2.year = $2"
		args = append(args, year)
	}

	query := fmt.Sprintf(`
SELECT e2.event_id,
       CASE WHEN override.event_id IS NOT NULL THEN 'event' ELSE 'group' END as level,
       CASE WHEN override.event_id IS NOT NULL THEN override.tier ELSE grp.tier END as tier,
       grp.tier as group_tier,
       CASE WHEN override.event_id IS NOT NULL THEN true ELSE false END as is_override
FROM events e2
JOIN starred_events grp ON grp.email = $1 AND grp.level = 'group'
JOIN events e1 ON grp.event_id = e1.event_id 
    AND e1.year = e2.year 
    AND e1.short_category = e2.short_category 
    AND e1.title = e2.title 
    AND e1.short_description = e2.short_description
LEFT JOIN starred_events override ON override.email = $1 AND override.event_id = e2.event_id AND override.level = 'event'
WHERE e2.active %s
ORDER BY e2.event_id`, yearFilter)

	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var starred StarredEvent
		err = rows.Scan(&starred.EventId, &starred.Level, &starred.Tier, &starred.GroupTier, &starred.IsOverride)
		if err != nil {
			return nil, err
		}
		starredEvents.StarredEvents = append(starredEvents.StarredEvents, starred)
	}

	return &starredEvents, nil
}

func LoadOrCreateUser(db *sql.DB, email string) (*User, error) {
	email = strings.ToLower(email)
	rows, err := db.Query(`
SELECT 
		email, 
		CASE WHEN length(display_name) > 0
            THEN display_name
            ELSE split_part(email, '@', 1)
            END,
		COALESCE(gencon_name, ''),
		COALESCE(gencon_id, ''),
		COALESCE(gencon_email, '')
FROM users
WHERE email=$1
`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var user *User
	if rows.Next() {
		var loadedUser User
		if err := rows.Scan(
			&loadedUser.Email,
			&loadedUser.DisplayName,
			&loadedUser.GenconName,
			&loadedUser.GenconId,
			&loadedUser.GenconEmail,
		); err != nil {
			log.Fatalf("Error loading user %v", err)
		} else {
			user = &loadedUser
		}
	}

	if user == nil {
		// Time to create a user
		user = &User{
			Email:       email,
			DisplayName: strings.Split(email, "@")[0],
		}

		_, err := db.Exec("INSERT INTO users(email, display_name) VALUES ($1, $2)",
			user.Email, user.DisplayName)
		if err != nil {
			log.Fatalf("Error creating user, %v", user)
		}
	}

	return user, nil
}
func GetWishlistConstraints(db *sql.DB, email string) ([]WishlistConstraint, error) {
	email = strings.ToLower(email)

	// Check if already initialized
	var initialized bool
	err := db.QueryRow("SELECT wishlist_constraints_initialized FROM users WHERE email = $1", email).Scan(&initialized)
	if err != nil {
		return nil, err
	}

	if !initialized {
		// Create default entry
		defaultConstraint := WishlistConstraint{
			DayOfWeek:   -1,
			StartHour:   23,
			StartMinute: 0,
			EndHour:     6,
			EndMinute:   0,
		}
		err = UpdateWishlistConstraints(db, email, []WishlistConstraint{defaultConstraint})
		if err != nil {
			return nil, err
		}
		return []WishlistConstraint{defaultConstraint}, nil
	}

	rows, err := db.Query(`
		SELECT COALESCE(day_of_week, -1), start_hour, start_minute, end_hour, end_minute, min_duration_minutes 
		FROM user_wishlist_constraints 
		WHERE email = $1`, email)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []WishlistConstraint
	for rows.Next() {
		var c WishlistConstraint
		if err := rows.Scan(&c.DayOfWeek, &c.StartHour, &c.StartMinute, &c.EndHour, &c.EndMinute, &c.MinDurationMinutes); err != nil {
			return nil, err
		}
		constraints = append(constraints, c)
	}

	return constraints, nil
}

func UpdateWishlistConstraints(db *sql.DB, email string, constraints []WishlistConstraint) error {
	email = strings.ToLower(email)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Mark as initialized
	_, err = tx.Exec("UPDATE users SET wishlist_constraints_initialized = TRUE WHERE email = $1", email)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM user_wishlist_constraints WHERE email = $1", email)
	if err != nil {
		return err
	}

	for _, c := range constraints {
		_, err = tx.Exec(`
			INSERT INTO user_wishlist_constraints (email, day_of_week, start_hour, start_minute, end_hour, end_minute, min_duration_minutes)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			email, c.DayOfWeek, c.StartHour, c.StartMinute, c.EndHour, c.EndMinute, c.MinDurationMinutes)
		if err != nil {
			return err
		}
	}

	// Mark wishlist as dirty
	_, err = tx.Exec("UPDATE users SET wishlist_dirty = TRUE, wishlist_updated_at = NOW() WHERE email = $1", email)
	if err != nil {
		return err
	}

	return tx.Commit()
}

type WishlistCacheItem struct {
	EventId   string
	Rank      int
	Status    string
	Reasoning []string
	Score     float64
}

func GetWishlistCache(db *sql.DB, email string, year int) ([]WishlistCacheItem, bool, time.Time, error) {
	email = strings.ToLower(email)
	var dirty bool
	var updatedAt time.Time
	err := db.QueryRow("SELECT wishlist_dirty, wishlist_updated_at FROM users WHERE email = $1", email).Scan(&dirty, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, true, time.Time{}, nil
		}
		return nil, false, time.Time{}, err
	}
	if dirty {
		return nil, true, updatedAt, nil
	}

	rows, err := db.Query(`
		SELECT event_id, rank, status, reasoning, score 
		FROM user_wishlist_cache 
		WHERE email = $1 AND year = $2 
		ORDER BY rank ASC`, email, year)
	if err != nil {
		return nil, false, time.Time{}, err
	}
	defer rows.Close()

	var cache []WishlistCacheItem
	for rows.Next() {
		var item WishlistCacheItem
		if err := rows.Scan(&item.EventId, &item.Rank, &item.Status, pq.Array(&item.Reasoning), &item.Score); err != nil {
			return nil, false, time.Time{}, err
		}
		cache = append(cache, item)
	}
	return cache, false, updatedAt, nil
}

func SaveWishlistCache(db *sql.DB, email string, year int, items []WishlistCacheItem, updatedAt time.Time) error {
	email = strings.ToLower(email)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec("DELETE FROM user_wishlist_cache WHERE email = $1 AND year = $2", email, year)
	if err != nil {
		return err
	}

	for _, item := range items {
		_, err = tx.Exec(`
			INSERT INTO user_wishlist_cache (email, year, event_id, rank, status, reasoning, score)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			email, year, item.EventId, item.Rank, item.Status, pq.Array(item.Reasoning), item.Score)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec("UPDATE users SET wishlist_dirty = FALSE WHERE email = $1 AND wishlist_updated_at = $2", email, updatedAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}
