package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/lib/pq"
)

type User struct {
	Email       string
	DisplayName string
}

type StarredEvent struct {
	EventId string
	Level   string // "group" or "event"
}

type UserStarredEvents struct {
	Email         string
	StarredEvents []StarredEvent
}

func UpdateDisplayName(db *sql.DB, email string, displayName string) error {
	email = strings.ToLower(email)
	_, err := db.Exec("UPDATE users SET display_name = $1 WHERE email = $2", displayName, email)
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
    ARRAY_AGG(se.event_id) event_ids
FROM starred_events se 
     JOIN events e ON e.event_id = se.event_id
WHERE se.email = $1
  AND e.year = $2
  AND e.active
GROUP BY e.cluster_key, day_of_week
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
		var dayOfWeek string
		err = rows.Scan(&dayOfWeek, pq.Array(&eventIds))
		if err != nil {
			return nil, err
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

		for _, event := range dayGroupEvents[1:] {
			if event.StartTime.After(cluster.EndTime) {
				groupedEvents = append(groupedEvents, cluster)
				cluster = newClusterForEvent(event)
			} else if event.EndTime.After(cluster.EndTime) {
				cluster.EndTime = event.EndTime
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
        AND (
          (se.level = 'event' AND e1.event_id = e2.event_id)
          OR
          (se.level = 'group' AND e1.year = e2.year AND e1.short_category = e2.short_category AND e1.title = e2.title AND e1.short_description = e2.short_description)
        )
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
        AND (
          (se.level = 'event' AND e1.event_id = e2.event_id)
          OR
          (se.level = 'group' AND e1.year = e2.year AND e1.short_category = e2.short_category AND e1.title = e2.title AND e1.short_description = e2.short_description)
        )
  )
ORDER BY e2.start_time`, fields), userEmail, year)

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

func UpdateStarredEvent(db *sql.DB, email string, eventId string, starGroup bool, add bool) (*UserStarredEvents, error) {
	return updateStarredEventInternal(db, email, eventId, starGroup, add, true)
}

func UpdateStarredEventMinimal(db *sql.DB, email string, eventId string, starGroup bool, add bool) (*UserStarredEvents, error) {
	return updateStarredEventInternal(db, email, eventId, starGroup, add, false)
}

func updateStarredEventInternal(db *sql.DB, email string, eventId string, starGroup bool, add bool, fullResponse bool) (*UserStarredEvents, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	if starGroup {
		// Delete all similar events first, regardless
		_, err = tx.Exec(`
DELETE FROM starred_events s
WHERE s.email = $1
  AND s.event_id in (
	  SELECT e2.event_id
	  FROM events e1 join events e2 on e1.year = e2.year
          AND e1.short_category = e2.short_category
	      AND e1.title = e2.title
          AND e1.short_description = e2.short_description
	  WHERE e1.event_id = $2
  )
`, email, eventId)

		if err == nil && add {
			// insert via select related ids
			_, err = tx.Exec(`
INSERT INTO starred_events(email, event_id, level)
SELECT $1, e2.event_id, 'group'
FROM events e1 join events e2 on e1.year = e2.year
    AND e1.short_category = e2.short_category
    AND e1.title = e2.title   
    AND e1.short_description = e2.short_description
WHERE e1.event_id = $2
ON CONFLICT DO NOTHING
`, email, eventId)
		}
	} else if add {
		// insert one record
		_, err = tx.Exec(`
INSERT INTO starred_events(email, event_id, level)
VALUES ($1, $2, 'event')
ON CONFLICT DO NOTHING
`, email, eventId)
	} else {
		// unstar individual session
		// 1. Demote any group star for this cluster to individual stars
		_, err = tx.Exec(`
UPDATE starred_events
SET level = 'event'
WHERE email = $1
  AND level = 'group'
  AND event_id IN (
    SELECT e2.event_id
    FROM events e1 JOIN events e2 ON e1.year = e2.year
          AND e1.short_category = e2.short_category
          AND e1.title = e2.title
          AND e1.short_description = e2.short_description
    WHERE e1.event_id = $2
  )
`, email, eventId)
		if err != nil {
			tx.Rollback()
			return nil, err
		}

		// 2. Delete the specific one
		_, err = tx.Exec(`
DELETE FROM starred_events s
WHERE s.email = $1
  AND s.event_id = $2
`, email, eventId)
	}

	if err != nil {
		tx.Rollback()
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
				EventId: eventId,
				Level:   level,
			}},
		}, nil
	}

	starredEvents, err := fetchStarredInternal(tx, email, 0) // 0 means all years
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
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
	return err
}

func BulkStarEvents(db *sql.DB, email string, year int, eventIds []string, overwrite bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
		stmt, err := tx.Prepare(pq.CopyIn("starred_events", "email", "event_id", "level"))
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, id := range eventIds {
			_, err = stmt.Exec(email, id, "event")
			if err != nil {
				return err
			}
		}

		_, err = stmt.Exec()
		if err != nil {
			return err
		}
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
}

func fetchStarredInternal(q queryable, email string, year int) (*UserStarredEvents, error) {
	starredEvents := UserStarredEvents{
		Email: email,
	}

	yearFilter := ""
	args := []interface{}{email}
	if year > 0 {
		yearFilter = " AND e1.year = $2"
		args = append(args, year)
	}

	query := fmt.Sprintf(`
SELECT DISTINCT e2.event_id, 
       CASE WHEN se.level = 'group' THEN 'group' ELSE 'event' END as level
FROM starred_events se
JOIN events e1 ON se.event_id = e1.event_id
JOIN events e2 ON (
    (se.level = 'event' AND e1.event_id = e2.event_id)
    OR
    (se.level = 'group' AND e1.year = e2.year AND e1.short_category = e2.short_category AND e1.title = e2.title AND e1.short_description = e2.short_description)
)
WHERE se.email = $1 %s AND e2.active`, yearFilter)

	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var starred StarredEvent
		err = rows.Scan(&starred.EventId, &starred.Level)
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
            END
FROM users
WHERE email=$1
`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var user *User
	for rows.Next() {
		var loadedUser User
		if err := rows.Scan(
			&loadedUser.Email,
			&loadedUser.DisplayName,
		); err != nil {
			log.Fatalf("Error loading user %v", err)
		} else {
			user = &loadedUser
		}

		break
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
