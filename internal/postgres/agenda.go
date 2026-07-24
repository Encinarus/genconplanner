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
SELECT %s, true, o.id, COALESCE(override.tier, grp.tier) as tier
FROM events e2
LEFT JOIN (
    SELECT lower(alias) as lower_alias, MAX(id) as id
    FROM orgs
    GROUP BY lower(alias)
) o ON o.lower_alias = lower(e2.org_group)
LEFT JOIN starred_events grp ON grp.email = $1 AND grp.level = 'group'
LEFT JOIN events e1 ON grp.event_id = e1.event_id 
    AND e1.year = e2.year 
    AND COALESCE(e1.short_category, '') = COALESCE(e2.short_category, '') 
    AND COALESCE(e1.title, '') = COALESCE(e2.title, '') 
    AND COALESCE(e1.short_description, '') = COALESCE(e2.short_description, '')
LEFT JOIN starred_events override ON override.email = $1 AND override.event_id = e2.event_id AND override.level = 'event'
WHERE e2.active 
  AND e2.year = $2
  AND (override.event_id IS NOT NULL OR (grp.event_id IS NOT NULL AND e1.event_id IS NOT NULL))
  AND COALESCE(override.tier, grp.tier) != 'not_interested'
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
