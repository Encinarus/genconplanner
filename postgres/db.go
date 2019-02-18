package postgres

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/Encinarus/genconplanner/events"
	"github.com/lib/pq"
	"log"
	"strings"
	"time"
)

var dbConnectString = flag.String("db", "", "postgres connect string")

func OpenDb() (*sql.DB, error) {
	return sql.Open("postgres", *dbConnectString)
}

type CategorySummary struct {
	Name  string
	Code  string
	Count int
}

type EventGroup struct {
	Name          string
	EventId       string
	Description   string
	ShortCategory string
	GameSystem    string
	Count         int
	WedTickets    int
	ThursTickets  int
	FriTickets    int
	SatTickets    int
	SunTickets    int
	TotalTickets  int
}

func rowToGroup(rows *sql.Rows) (*EventGroup, error) {
	var group EventGroup
	if err := rows.Scan(
		&group.Name,
		&group.Description,
		&group.ShortCategory,
		&group.GameSystem,
		// Aggregate fields
		&group.Count,
		&group.EventId,
		&group.WedTickets,
		&group.ThursTickets,
		&group.FriTickets,
		&group.SatTickets,
		&group.SunTickets,
	); err != nil {
		return nil, err
	}
	group.TotalTickets = group.WedTickets + group.ThursTickets + group.FriTickets + group.SatTickets + group.SunTickets
	return &group, nil
}

func LoadEventGroups(db *sql.DB, cat string, year int) ([]*EventGroup, error) {
	rows, err := db.Query(`
SELECT 
       title, 
       short_description,
       short_category,
       game_system,
       count(1),
       min(event_id),
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 3 THEN tickets_available ELSE 0 END) as wednesday_tickets,
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 4 THEN tickets_available ELSE 0 END) as thursday_tickets,
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 5 THEN tickets_available ELSE 0 END) as friday_tickets,
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 6 THEN tickets_available ELSE 0 END) as saturday_tickets,
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 0 THEN tickets_available ELSE 0 END) as sunday_tickets
FROM event
WHERE active and year=$1 and short_category=$2
GROUP BY 
         title,
         short_description,
         short_category,
         game_system,
         cluster_key
ORDER BY sum(tickets_available) > 0 desc
`, year, cat)
	if err != nil {
		return nil, err
	}

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

func LoadCategorySummary(db *sql.DB, year int) ([]*CategorySummary, error) {
	rows, err := db.Query(`
SELECT event_type, COUNT(1)
FROM event
where active and year = $1
GROUP BY event_type
ORDER BY event_type ASC`, year)

	if err != nil {
		return nil, err
	}
	countsPerCategory := make([]*CategorySummary, 0)
	for rows.Next() {
		var summary CategorySummary

		if err = rows.Scan(&summary.Name, &summary.Count); err != nil {
			return nil, err
		}
		summary.Code = strings.Split(summary.Name, " ")[0]
		countsPerCategory = append(countsPerCategory, &summary)
	}
	return countsPerCategory, nil
}

func LoadSimilarEvents(db *sql.DB, eventId string) ([]*events.GenconEvent, error) {
	// Might be slight overkill ensuring that the year matches, but
	// folks could submit the same event two years in a row with the same
	// description, making it cluster the same.
	year := events.YearFromEvent(eventId)

	fields := "e1." + strings.Join(eventFields(), ", e1.")
	rows, err := db.Query(fmt.Sprintf(`
SELECT %s
FROM event e1 join event e2 on e1.cluster_key = e2.cluster_key
WHERE e2.event_id = $1
  AND e1.year = $2`, fields), eventId, year)

	if err != nil {
		return nil, err
	}

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

type ParsedQuery struct {
	// TODO(alek): make a significantly more robust query parser
	// add exact match on fields,
	TextQueries []string
	Year        int
}

func FindEvents(db *sql.DB, query *ParsedQuery) ([]*EventGroup, error) {
	tsquery := strings.Join(query.TextQueries, " & ")

	// We get groups that have tickets first, then within
	// that, we rank by how good a match the query was
	loadedEvents := make([]*EventGroup, 0)
	if len(tsquery) > 0 {
		rows, err := db.Query(`
SELECT 
       title, 
       short_description, 
       short_category,    
       game_system,
       count(1),
       min(event_id),
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 3 THEN tickets_available ELSE 0 END) as wednesday_tickets,
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 4 THEN tickets_available ELSE 0 END) as thursday_tickets,
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 5 THEN tickets_available ELSE 0 END) as friday_tickets,
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 6 THEN tickets_available ELSE 0 END) as saturday_tickets,
       sum(CASE WHEN EXTRACT(DOW FROM start_time) = 0 THEN tickets_available ELSE 0 END) as sunday_tickets
FROM event, to_tsquery($1) q
WHERE active and cluster_key @@ q and year = $2
GROUP BY 
         title, 
         short_description,
         short_category, 
         game_system,
         cluster_key,
         ts_rank(title_tsv, q), ts_rank(cluster_key, q)
ORDER BY sum(tickets_available) > 0 desc, 
         ts_rank(title_tsv, q) desc,
         ts_rank(cluster_key, q) desc`, tsquery, query.Year)

		if err != nil {
			return nil, err
		}

		// Load all the events
		for rows.Next() {
			group, err := rowToGroup(rows)
			if err != nil {
				return nil, err
			}

			loadedEvents = append(loadedEvents, group)
		}
	}

	return loadedEvents, nil
}

func loadEventIds(tx *sql.Tx, year int) (map[string]time.Time, map[string]time.Time, error) {
	// load all events: ids + last update time
	rows, err := tx.Query(`
SELECT event_id, active, last_modified
FROM event
WHERE year=$1`, year)
	if err != nil {
		return nil, nil, err
	}

	var activeEvents map[string]time.Time
	var inactiveEvents map[string]time.Time
	activeEvents = make(map[string]time.Time)
	inactiveEvents = make(map[string]time.Time)

	for rows.Next() {
		var id string
		var active bool
		var updateTime time.Time

		err := rows.Scan(&id, &active, &updateTime)
		if err != nil {
			return nil, nil, err
		}

		if active {
			activeEvents[id] = updateTime
		} else {
			inactiveEvents[id] = updateTime
		}
	}

	return activeEvents, inactiveEvents, nil
}

func bulkDelete(tx *sql.Tx, deletedEvents []string) error {
	// Deletes aren't true deletes, we mark them as inactive
	batchSize := 100
	for len(deletedEvents) > 0 {
		if len(deletedEvents) < batchSize {
			batchSize = len(deletedEvents)
		}
		batch := make([]string, 0, batchSize)
		for _, eventId := range deletedEvents[0:batchSize:batchSize] {
			batch = append(batch, "'"+eventId+"'")
		}

		deletedEvents = deletedEvents[batchSize:]
		updateStatement := fmt.Sprintf(
			"UPDATE event SET active = FALSE WHERE event_id in (%s)",
			strings.Join(batch, ","))

		_, err := tx.Exec(updateStatement)

		if err != nil {
			log.Printf("Error on processing event: %s %v", batch, err.(pq.PGError))
			return err
		}
	}
	return nil
}

func BulkUpdateEvents(tx *sql.Tx, parsedEvents []*events.GenconEvent) error {
	year := parsedEvents[0].Year
	activeEvents, inactiveEvents, err := loadEventIds(tx, year)
	persistedEvents := make(map[string]time.Time, len(activeEvents)+len(inactiveEvents))
	for id, updateTime := range activeEvents {
		persistedEvents[id] = updateTime
	}
	for id, updateTime := range inactiveEvents {
		persistedEvents[id] = updateTime
	}

	if err != nil {
		return err
	}
	log.Printf("Loaded %d Rows\n", len(persistedEvents))

	var newEvents []*events.GenconEvent
	var updatedEvents []*events.GenconEvent

	latestUpdate := parsedEvents[0].LastModified
	for _, parsedEvent := range parsedEvents {
		if updateTime, found := persistedEvents[parsedEvent.EventId]; found {
			if updateTime.Before(parsedEvent.LastModified) {
				updatedEvents = append(updatedEvents, parsedEvent)
			}
			delete(activeEvents, parsedEvent.EventId)
		} else {
			newEvents = append(newEvents, parsedEvent)
		}
		if latestUpdate.Before(parsedEvent.LastModified) {
			latestUpdate = parsedEvent.LastModified
		}
	}

	// Any remaining active events should be deleted
	deletedEvents := make([]string, 0, len(activeEvents))
	for event := range activeEvents {
		deletedEvents = append(deletedEvents, event)
	}

	log.Printf("Inserting %d events\n", len(newEvents))
	log.Printf("Updating %d events\n", len(updatedEvents))
	log.Printf("Deleting %d events\n", len(deletedEvents))

	err = bulkInsert(tx, newEvents)
	if err != nil {
		return err
	}
	err = bulkUpdate(tx, updatedEvents)
	if err != nil {
		return err
	}
	err = bulkDelete(tx, deletedEvents)
	return err
}

func rangeSlice(min, max int) []interface{} {
	a := make([]interface{}, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func eventFields() []string {
	return []string{
		"event_id",
		"year",
		"active",
		"org_group",
		"title",
		"short_description",
		"long_description",
		"event_type",
		"game_system",
		"rules_edition",
		"min_players",
		"max_players",
		"age_required",
		"experience_required",
		"materials_provided",
		"start_time",
		"duration",
		"end_time",
		"gm_names",
		"website",
		"email",
		"tournament",
		"round_number",
		"total_rounds",
		"min_play_time",
		"attendee_registration",
		"cost",
		"location",
		"room_name",
		"table_number",
		"special_category",
		"tickets_available",
		"last_modified",
		"short_category",
	}
}

func eventToDbFields(event *events.GenconEvent) []interface{} {

	return []interface{}{
		event.EventId,
		event.Year,
		event.Active,
		event.Group,
		event.Title,
		event.ShortDescription,
		event.LongDescription,
		event.EventType,
		event.GameSystem,
		event.RulesEdition,
		event.MinPlayers,
		event.MaxPlayers,
		event.AgeRequired,
		event.ExperienceRequired,
		event.MaterialsProvided,
		event.StartTime,
		event.Duration,
		event.EndTime,
		event.GMNames,
		event.Website,
		event.Email,
		event.Tournament,
		event.RoundNumber,
		event.TotalRounds,
		event.MinPlayTime,
		event.AttendeeRegistration,
		event.Cost,
		event.Location,
		event.RoomName,
		event.TableNumber,
		event.SpecialCategory,
		event.TicketsAvailable,
		event.LastModified,
		event.ShortCategory,
	}
}

func scanEvent(row *sql.Rows) (*events.GenconEvent, error) {
	var event events.GenconEvent

	err := row.Scan(
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
		&event.ShortCategory)

	return &event, err
}

func bulkUpdate(tx *sql.Tx, updatedRows []*events.GenconEvent) error {
	eventFields := eventFields()
	numEventFields := len(eventFields)

	for _, row := range updatedRows {
		whereClause := fmt.Sprintf(
			"(%s) = %s",
			strings.Join(eventFields, ", "),
			fmt.Sprintf(
				"($%d"+strings.Repeat(", $%d", numEventFields-1)+")",
				rangeSlice(1, numEventFields)...))
		updateStatement := fmt.Sprintf(
			"UPDATE event SET %s WHERE event_id='%s'",
			whereClause,
			row.EventId)

		valueArgs := eventToDbFields(row)
		_, err := tx.Exec(updateStatement, valueArgs...)

		if err != nil {
			log.Printf("Error on updating event: %v %v", row, err.(pq.PGError))
			return err
		}
	}

	return nil

}

func bulkInsert(tx *sql.Tx, newRows []*events.GenconEvent) error {
	batchSize := 100

	eventFields := eventFields()
	numEventFields := len(eventFields)

	for len(newRows) > 0 {
		if batchSize > len(newRows) {
			// This is the final batch
			batchSize = len(newRows)
		}
		batch := newRows[0:batchSize:batchSize]
		newRows = newRows[batchSize:]

		valueStrings := make([]string, 0, len(batch))
		valueArgs := make([]interface{}, 0, len(batch)*numEventFields)
		for i, row := range batch {
			valueStrings = append(
				valueStrings,
				fmt.Sprintf(
					"( $%d "+strings.Repeat(",$%d", numEventFields-1)+" )",
					rangeSlice(i*numEventFields+1, i*numEventFields+numEventFields)...))
			valueArgs = append(valueArgs, eventToDbFields(row)...)
		}
		insertStatement := fmt.Sprintf(
			"INSERT INTO event (%s) VALUES %s",
			strings.Join(eventFields, ","),
			strings.Join(valueStrings, ","))
		_, err := tx.Exec(insertStatement, valueArgs...)

		if err != nil {
			log.Printf("Error on processing event: %s %v", batch, err.(pq.PGError))
			return err
		}
	}

	return nil
}
