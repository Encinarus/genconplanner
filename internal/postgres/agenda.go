package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Encinarus/genconplanner/internal/events"
)

type AgendaEntry struct {
	Event *events.GenconEvent
	Tier  string
}

func LoadAgenda(db *sql.DB, userEmail string, year int) ([]*AgendaEntry, error) {
	fields := "e2." + strings.Join(eventFields(), ", e2.")
	rows, err := db.Query(fmt.Sprintf(`
WITH user_stars AS (
    SELECT se.event_id, se.level, se.tier, e.year, COALESCE(e.short_category, '') as short_category, COALESCE(e.title, '') as title, COALESCE(e.short_description, '') as short_description
    FROM starred_events se
    JOIN events e ON se.event_id = e.event_id
    WHERE se.email = $1 AND e.year = $2 AND e.active
)
SELECT %s, true, o.id, COALESCE(override.tier, grp.tier) as tier
FROM events e2
LEFT JOIN (
    SELECT lower(alias) as lower_alias, MAX(id) as id
    FROM orgs
    GROUP BY lower(alias)
) o ON o.lower_alias = lower(e2.org_group)
LEFT JOIN user_stars grp ON grp.level = 'group'
    AND e2.year = grp.year 
    AND COALESCE(e2.short_category, '') = grp.short_category 
    AND COALESCE(e2.title, '') = grp.title 
    AND COALESCE(e2.short_description, '') = grp.short_description
LEFT JOIN user_stars override ON override.level = 'event' AND override.event_id = e2.event_id
WHERE e2.active 
  AND e2.year = $2
  AND (grp.event_id IS NOT NULL OR override.event_id IS NOT NULL)
  AND COALESCE(override.tier, grp.tier, 'not_interested') != 'not_interested'
ORDER BY e2.start_time`, fields), userEmail, year)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agenda := make([]*AgendaEntry, 0)
	for rows.Next() {
		var event events.GenconEvent
		var starred bool
		var orgId sql.NullInt64
		var tier string

		// This is a bit brittle if eventFields changes, but it's the most
		// direct way to get the data without modifying scanEvent globally.
		err := rows.Scan(
			&event.EventId,
			&event.Year,
			&event.Active,
			&event.Group,
			&event.Title,
			&event.ShortDescription,
			&event.LongDescription,
			&event.EventType,
			&event.GameSystem,
			&event.RulesEdition,
			&event.MinPlayers,
			&event.MaxPlayers,
			&event.AgeRequired,
			&event.ExperienceRequired,
			&event.MaterialsProvided,
			&event.StartTime,
			&event.Duration,
			&event.EndTime,
			&event.GMNames,
			&event.Website,
			&event.Email,
			&event.Tournament,
			&event.RoundNumber,
			&event.TotalRounds,
			&event.MinPlayTime,
			&event.AttendeeRegistration,
			&event.Cost,
			&event.Location,
			&event.RoomName,
			&event.TableNumber,
			&event.SpecialCategory,
			&event.TicketsAvailable,
			&event.LastModified,
			&event.ShortCategory,
			&starred,
			&orgId,
			&tier,
		)
		if err != nil {
			return nil, err
		}
		event.IsStarred = starred
		if orgId.Valid {
			event.OrgId = orgId.Int64
		}

		agenda = append(agenda, &AgendaEntry{
			Event: events.NormalizeEvent(&event),
			Tier:  tier,
		})
	}
	return agenda, nil
}
