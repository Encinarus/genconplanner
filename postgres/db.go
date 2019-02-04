package postgres

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/Encinarus/genconplanner/events"
	"github.com/lib/pq"
	"log"
	"strings"
	"time"
)

type parsedQuery struct {
	// TODO(alek): make a significantly more robust query parser
	// add exact match on fields,
	textQueries []string
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

func FindEvents(db *sql.DB, searchQuery string) []*events.GenconEvent {
	// Preprocess, removing symbols which are used in tsquery
	searchQuery = strings.Replace(searchQuery, "!", "", -1)
	searchQuery = strings.Replace(searchQuery, "&", "", -1)
	searchQuery = strings.Replace(searchQuery, "(", "", -1)
	searchQuery = strings.Replace(searchQuery, ")", "", -1)
	searchQuery = strings.Replace(searchQuery, "|", "", -1)

	queryReader := csv.NewReader(bytes.NewBufferString(searchQuery))
	queryReader.Comma = ' '

	splitQuery, _ := queryReader.Read()

	query := parsedQuery{}
	// TODO(alek): consider adding a db field "searchable_text" rather than relying
	// the trigger across many fields. Then exact matches do like vs that, while fuzzy
	// matches go against the ts_vector column
	for _, term := range splitQuery {
		invertTerm := false
		if strings.HasPrefix(term, "-") {
			log.Println("Negated term:", term)
			term = strings.TrimLeft(term, "-")
			invertTerm = true
		}
		if strings.ContainsAny(term, ":<>=-~") {
			// TODO(alek) Handle key:value searches
			// : and = work as equals
			// < > compare for dates or num tickets
			// ~ is for checking if the string is in a field
			continue
		}

		// Now remove remaining symbols we want to allow in field-specific
		// searches, but not in the general text search
		term = strings.Replace(term, "<", "", -1)
		term = strings.Replace(term, ">", "", -1)
		term = strings.Replace(term, "=", "", -1)
		term = strings.Replace(term, "-", "", -1)
		term = strings.Replace(term, "~", "", -1)
		term = strings.TrimSpace(term)
		if len(term) == 0 {
			continue
		}
		if invertTerm {
			term = "!" + term
		}
		query.textQueries = append(query.textQueries, term)
	}
	tsquery := strings.Join(query.textQueries, " & ")

	// We get groups that have tickets first, then within
	// that, we rank by how good a match the query was
	loadedEvents := make([]*events.GenconEvent, 0)
	if len(tsquery) > 0 {
		rawQuery := fmt.Sprintf(`
SELECT %s
FROM event, to_tsquery('%s') q
WHERE active 
  AND tsv @@ q
ORDER BY tickets_available > 0 desc, ts_rank(tsv, q) desc
`, strings.Join(eventFields(), ", "), tsquery)
		fmt.Println(rawQuery)
		rows, err := db.Query(rawQuery)
		if err != nil {
			log.Fatalf("Unable to execute query %s, %v", searchQuery, err)
		}

		// Load all the events
		for rows.Next() {
			event, err := scanEvent(rows)
			if err != nil {
				log.Fatalf("Unable to load event after query, %v", err)
			}

			loadedEvents = append(loadedEvents, event)
		}
	}

	return loadedEvents
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
			batch = append(batch, "'" + eventId + "'")
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
	persistedEvents := make(map[string]time.Time, len(activeEvents) + len(inactiveEvents))
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
	}
}

func eventToDbFields(event *events.GenconEvent) []interface{} {
	return []interface{} {
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
	}
}

func scanEvent(row *sql.Rows) (*events.GenconEvent, error){
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
		&event.LastModified)

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
				"($%d" + strings.Repeat(", $%d", numEventFields - 1) + ")",
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
		valueArgs := make([]interface{}, 0, len(batch) * numEventFields)
		for i, row := range batch {
			valueStrings = append(
				valueStrings,
				fmt.Sprintf(
					"( $%d " + strings.Repeat(",$%d", numEventFields - 1) +" )",
					rangeSlice(i*numEventFields + 1, i*numEventFields + numEventFields)...))
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

