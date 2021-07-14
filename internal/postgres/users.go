package postgres

import (
	"database/sql"
	"fmt"
	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/lib/pq"
	"log"
	"sort"
	"strings"
)

type User struct {
	Email       string
	DisplayName string
}


type UserStarredEvents struct {
	Email         string
	StarredEvents []string
}

func (u *User) UpdateInfo(db *sql.DB, displayName string) error {
	u.DisplayName = displayName

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Cleanup transaction!
	defer func() { CleanupTransaction(err, tx) }()

	return nil
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

		groupedEvents = append(groupedEvents, cluster)
	}

	log.Printf("Returning %v groups", len(groupedEvents))
	return groupedEvents, nil
}

func LoadStarredEvents(db *sql.DB, userEmail string, year int) ([]*events.GenconEvent, error) {
	fields := "e1." + strings.Join(eventFields(), ", e1.")
	rows, err := db.Query(fmt.Sprintf(`
SELECT %s, true
FROM events e1 
     JOIN starred_events se ON se.event_id = e1.event_id
WHERE se.email = $1
  AND e1.year = $2
  AND e1.active
ORDER BY e1.start_time`, fields), userEmail, year)

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

func UpdateStarredEvent(db *sql.DB, email string, eventId string, related bool, add bool) (*UserStarredEvents, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	if related {
		// Delete all similar events first, regardless
		_, err = tx.Exec(`
DELETE FROM starred_events s
WHERE s.email = $1
  AND s.event_id in (
	  SELECT e2.event_id
	  FROM events e1 join events e2 on e1.year = e2.year
          AND e1.short_category = e2.short_category
	      AND e1.title = e2.title
          AND e1.cluster_key = e2.cluster_key
	  WHERE e1.event_id = $2
  )
`, email, eventId)

		if err == nil && add {
			// insert via select related ids
			_, err = tx.Exec(`
INSERT INTO starred_events(email, event_id)
SELECT $1, e2.event_id
FROM events e1 join events e2 on e1.year = e2.year
    AND e1.short_category = e2.short_category
    AND e1.title = e2.title   
    AND e1.cluster_key = e2.cluster_key
WHERE e1.event_id = $2
ON CONFLICT DO NOTHING
`, email, eventId)
		}
	} else if add {
		// insert one record
		_, err = tx.Exec(`
INSERT INTO starred_events(email, event_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, email, eventId)
	} else {
		// delete record
		_, err = tx.Exec(`
DELETE FROM starred_events s
WHERE s.email = $1
  AND s.event_id = $2
`, email, eventId)
	}

	if err != nil {
		tx.Rollback()
		return nil, err
	} else {
		starredEvents := UserStarredEvents{
			Email: email,
		}

		err = tx.QueryRow(`
SELECT ARRAY(SELECT event_id
FROM starred_events
WHERE email = $1);
`, email).Scan(pq.Array(&starredEvents.StarredEvents))
		if err != nil {
			tx.Rollback()
			return nil, err
		} else {
			tx.Commit()
			return &starredEvents, nil
		}
	}
}

func GetStarredIds(db *sql.DB, email string) (*UserStarredEvents, error) {
	starredEvents := UserStarredEvents{
		Email: email,
	}

	err := db.QueryRow(`
SELECT ARRAY(SELECT event_id
FROM starred_events
WHERE email = $1);
`, email).Scan(pq.Array(&starredEvents.StarredEvents))
	if err != nil {
		return nil, err
	} else {
		return &starredEvents, nil
	}
}

func LoadOrCreateUser(db *sql.DB, email string) (*User, error) {
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
