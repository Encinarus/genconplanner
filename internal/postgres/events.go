package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/lib/pq"
)

type CalendarEventCluster struct {
	EventId          string
	Title            string
	StartTime        time.Time
	EndTime          time.Time
	GenconUrl        string
	PlannerUrl       string
	ShortCategory    string
	ShortDescription string
	SimilarCount     int
	Location         string
	RoomName         string
	TableNumber      string
}

func newClusterForEvent(event *events.GenconEvent) *CalendarEventCluster {
	return &CalendarEventCluster{
		EventId:          event.EventId,
		Title:            event.Title,
		StartTime:        event.StartTime,
		EndTime:          event.EndTime,
		GenconUrl:        event.GenconLink(),
		PlannerUrl:       event.PlannerLink(),
		ShortCategory:    event.ShortCategory,
		ShortDescription: event.ShortDescription,
		SimilarCount:     1,
		Location:         event.Location,
		RoomName:         event.RoomName,
		TableNumber:      event.TableNumber,
	}
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
	OrgGroup      string
	OrgId             int
	Count             int
	TotalTickets      int
	WedEvents         int
	WedTotalTickets   int
	WedTickets        int
	ThursEvents       int
	ThursTotalTickets int
	ThursTickets      int
	FriEvents         int
	FriTotalTickets   int
	FriTickets        int
	SatEvents         int
	SatTotalTickets   int
	SatTickets        int
	SunEvents         int
	SunTotalTickets   int
	SunTickets        int
}

type ParsedQuery struct {
	// TODO(alek): make a significantly more robust query parser
	// add exact match on fields,
	TextQueries     []string
	Year            int
	DaysOfWeek      map[string]bool
	RawQuery        string
	StartBeforeHour int
	StartAfterHour  int
	EndBeforeHour   int
	EndAfterHour    int
	OrgId           int
}

type SearchQuery struct {
	Year              int
	CategoryShortCode string
	MinWedTickets     int
	MinThuTickets     int
	MinFriTickets     int
	MinSatTickets     int
	MinSunTickets     int
	RawQuery          string
	OrgId             int
	OnlyFree          bool
	UserEmail         string
}

func rowToGroup(rows *sql.Rows) (*EventGroup, error) {
	var group EventGroup
	var title_rank float64
	var search_rank float64
	if err := rows.Scan(
		&group.EventId,
		&group.Name,
		&group.Description,
		&group.ShortCategory,
		&group.GameSystem,
		&group.OrgGroup,
		&group.OrgId,
		// Aggregate fields
		&group.Count,
		&group.TotalTickets,
		&group.WedEvents,
		&group.WedTotalTickets,
		&group.WedTickets,
		&group.ThursEvents,
		&group.ThursTotalTickets,
		&group.ThursTickets,
		&group.FriEvents,
		&group.FriTotalTickets,
		&group.FriTickets,
		&group.SatEvents,
		&group.SatTotalTickets,
		&group.SatTickets,
		&group.SunEvents,
		&group.SunTotalTickets,
		&group.SunTickets,
		&title_rank,
		&search_rank,
	); err != nil {
		return nil, err
	}
	return &group, nil
}

func SearchEvents(ctx context.Context, db *sql.DB, query SearchQuery) ([]*EventGroup, error) {
	results := make([]*EventGroup, 0)

	onlyFreeVal := 0
	if query.OnlyFree && len(query.UserEmail) > 0 {
		onlyFreeVal = 1
	}

	// Optional search terms should be incorporated into the WHERE clause as
	// AND (<term was omitted> OR <apply term>)
	rows, err := db.QueryContext(ctx, `
SELECT
	e.event_id,
	e.title, 
	e.short_description,
	e.short_category,
	e.game_system,
	e.org_group,
	o.id AS org_id,
	c.num_events,
	c.tickets_available,
	c.wed_events,
	c.wed_total_tickets,
	c.wed_tickets,
	c.thu_events,
	c.thu_total_tickets,
	c.thu_tickets,
	c.fri_events,
	c.fri_total_tickets,
	c.fri_tickets,
	c.sat_events,
	c.sat_total_tickets,
	c.sat_tickets,
	c.sun_events,
	c.sun_total_tickets,
	c.sun_tickets,
	0 as title_rank,
	0 as search_rank
FROM events e
JOIN (
    SELECT
        MIN(s.event_id) AS event_id,
        COUNT(*) AS num_events,
        SUM(s.tickets_available) AS tickets_available,
        COUNT(CASE WHEN s.day_of_week = 3 THEN 1 ELSE NULL END) as wed_events,
        SUM(CASE WHEN s.day_of_week = 3 THEN s.max_players ELSE 0 END) as wed_total_tickets,
        SUM(CASE WHEN s.day_of_week = 3 THEN s.tickets_available ELSE 0 END) as wed_tickets,
        COUNT(CASE WHEN s.day_of_week = 4 THEN 1 ELSE NULL END) as thu_events,
        SUM(CASE WHEN s.day_of_week = 4 THEN s.max_players ELSE 0 END) as thu_total_tickets,
        SUM(CASE WHEN s.day_of_week = 4 THEN s.tickets_available ELSE 0 END) as thu_tickets,
        COUNT(CASE WHEN s.day_of_week = 5 THEN 1 ELSE NULL END) as fri_events,
        SUM(CASE WHEN s.day_of_week = 5 THEN s.max_players ELSE 0 END) as fri_total_tickets,
        SUM(CASE WHEN s.day_of_week = 5 THEN s.tickets_available ELSE 0 END) as fri_tickets,
        COUNT(CASE WHEN s.day_of_week = 6 THEN 1 ELSE NULL END) as sat_events,
        SUM(CASE WHEN s.day_of_week = 6 THEN s.max_players ELSE 0 END) as sat_total_tickets,
        SUM(CASE WHEN s.day_of_week = 6 THEN s.tickets_available ELSE 0 END) as sat_tickets,
        COUNT(CASE WHEN s.day_of_week = 0 THEN 1 ELSE NULL END) as sun_events,
        SUM(CASE WHEN s.day_of_week = 0 THEN s.max_players ELSE 0 END) as sun_total_tickets,
        SUM(CASE WHEN s.day_of_week = 0 THEN s.tickets_available ELSE 0 END) as sun_tickets
    FROM events s
    WHERE s.active
      AND (LENGTH($1) = 0 OR s.short_category = $1)
      AND ($2 = 0 OR s.year = $2)
      AND ($3 = 0 OR (s.day_of_week = 3 AND s.tickets_available >= $3))
      AND ($4 = 0 OR (s.day_of_week = 4 AND s.tickets_available >= $4))
      AND ($5 = 0 OR (s.day_of_week = 5 AND s.tickets_available >= $5))
      AND ($6 = 0 OR (s.day_of_week = 6 AND s.tickets_available >= $6))
      AND ($7 = 0 OR (s.day_of_week = 0 AND s.tickets_available >= $7))
      AND (LENGTH($8) = 0 OR (s.search_key @@ websearch_to_tsquery('english', $8)))
      AND ($10 = 0 OR NOT EXISTS (
          SELECT 1 
          FROM starred_events se
          JOIN events pe ON se.event_id = pe.event_id
          WHERE se.email = $11 
            AND se.tier = 'purchased'
            AND pe.active
            AND s.start_time < pe.end_time
            AND s.end_time > pe.start_time
      ))
    GROUP BY s.cluster_id
) c ON e.event_id = c.event_id
JOIN (
    SELECT lower(alias) as alias, MAX(id) as id
    FROM orgs
    GROUP BY lower(alias)
) o ON o.alias = lower(e.org_group)
WHERE ($9 = 0 OR o.id = $9)
	`, query.CategoryShortCode, query.Year, query.MinWedTickets,
		query.MinThuTickets, query.MinFriTickets, query.MinSatTickets,
		query.MinSunTickets, query.RawQuery, query.OrgId,
		onlyFreeVal, query.UserEmail)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		group, err := rowToGroup(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, group)
	}

	return results, nil
}

func LoadEventGroupsForCategory(ctx context.Context, db *sql.DB, short_category string, year int) ([]*EventGroup, error) {
	rows, err := db.QueryContext(ctx, `
SELECT 
	e.event_id,
	e.title,
	e.short_description,
	e.short_category,
	e.game_system,
	e.org_group,
	o.id AS org_id,
	c.num_events,
	c.tickets_available,
	c.wed_events,
	c.wed_total_tickets,
	c.wed_tickets,
	c.thu_events,
	c.thu_total_tickets,
	c.thu_tickets,
	c.fri_events,
	c.fri_total_tickets,
	c.fri_tickets,
	c.sat_events,
	c.sat_total_tickets,
	c.sat_tickets,
	c.sun_events,
	c.sun_total_tickets,
	c.sun_tickets,
	0 as title_rank,
	0 as search_rank
FROM events e 
JOIN (
    SELECT 
        min(event_id) as event_id,
        count(active or null) as num_events,
        sum(tickets_available) as tickets_available,
        COUNT(CASE WHEN day_of_week = 3 THEN 1 ELSE NULL END) as wed_events,
        SUM(CASE WHEN day_of_week = 3 THEN max_players ELSE 0 END) as wed_total_tickets,
        SUM(CASE WHEN day_of_week = 3 THEN tickets_available ELSE 0 END) as wed_tickets,
        COUNT(CASE WHEN day_of_week = 4 THEN 1 ELSE NULL END) as thu_events,
        SUM(CASE WHEN day_of_week = 4 THEN max_players ELSE 0 END) as thu_total_tickets,
        SUM(CASE WHEN day_of_week = 4 THEN tickets_available ELSE 0 END) as thu_tickets,
        COUNT(CASE WHEN day_of_week = 5 THEN 1 ELSE NULL END) as fri_events,
        SUM(CASE WHEN day_of_week = 5 THEN max_players ELSE 0 END) as fri_total_tickets,
        SUM(CASE WHEN day_of_week = 5 THEN tickets_available ELSE 0 END) as fri_tickets,
        COUNT(CASE WHEN day_of_week = 6 THEN 1 ELSE NULL END) as sat_events,
        SUM(CASE WHEN day_of_week = 6 THEN max_players ELSE 0 END) as sat_total_tickets,
        SUM(CASE WHEN day_of_week = 6 THEN tickets_available ELSE 0 END) as sat_tickets,
        COUNT(CASE WHEN day_of_week = 0 THEN 1 ELSE NULL END) as sun_events,
        SUM(CASE WHEN day_of_week = 0 THEN max_players ELSE 0 END) as sun_total_tickets,
        SUM(CASE WHEN day_of_week = 0 THEN tickets_available ELSE 0 END) as sun_tickets
    FROM events
    WHERE active and year=$1 and short_category=$2
    GROUP BY cluster_id
) as c ON e.event_id = c.event_id
JOIN (
    SELECT lower(alias) as alias, MAX(id) as id
    FROM orgs
    GROUP BY lower(alias)
) o ON o.alias = lower(e.org_group)
WHERE e.year = $1
ORDER BY c.tickets_available > 0 desc, title`, year, short_category)
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

func LoadCategorySummary(db *sql.DB, year int) ([]*CategorySummary, error) {
	rows, err := db.Query(`
SELECT event_type, COUNT(1)
FROM events
where active and year = $1
GROUP BY event_type
ORDER BY event_type ASC`, year)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

func LoadSimilarEvents(ctx context.Context, db *sql.DB, eventId string, userEmail string) ([]*events.GenconEvent, error) {
	// Might be slight overkill ensuring that the year matches, but
	// folks could submit the same event two years in a row with the same
	// description, making it cluster the same.
	year := events.YearFromEvent(eventId)

	fields := "e1." + strings.Join(eventFields(), ", e1.")
	// #nosec G201
	raw_query := fmt.Sprintf(`
	SELECT distinct %s, se.event_id is not null, o.id
	FROM events e1 
		 JOIN events e2 on e1.cluster_id = e2.cluster_id
		 LEFT JOIN starred_events se ON se.event_id = e1.event_id AND se.email = $3
		 LEFT JOIN orgs o ON lower(o.alias) = lower(e1.org_group)
	WHERE e2.event_id = $1
	  AND e1.year = $2
	ORDER BY e1.start_time`, fields)
	rows, err := db.QueryContext(ctx, raw_query, eventId, year, userEmail)

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
		loadedEvents = append(loadedEvents, events.NormalizeEvent(event))
	}
	return loadedEvents, nil
}

func FindEvents(db *sql.DB, query *ParsedQuery) ([]*EventGroup, error) {
	var args []interface{}
	argIndex := 1

	args = append(args, query.Year)
	yearArg := fmt.Sprintf("$%d", argIndex)
	argIndex++

	innerFrom := "events"
	innerWhere := fmt.Sprintf("active AND year = %s", yearArg)
	if query.StartBeforeHour >= 0 {
		args = append(args, query.StartBeforeHour)
		innerWhere = fmt.Sprintf("%s AND EXTRACT(HOUR FROM start_time AT TIME ZONE 'EDT') <= $%d", innerWhere, argIndex)
		argIndex++
	}
	if query.StartAfterHour >= 0 {
		args = append(args, query.StartAfterHour)
		innerWhere = fmt.Sprintf("%s AND EXTRACT(HOUR FROM start_time AT TIME ZONE 'EDT') >= $%d", innerWhere, argIndex)
		argIndex++
	}
	if query.EndBeforeHour >= 0 {
		args = append(args, query.EndBeforeHour)
		innerWhere = fmt.Sprintf("%s AND EXTRACT(HOUR FROM end_time AT TIME ZONE 'EDT') <= $%d", innerWhere, argIndex)
		argIndex++
	}
	if query.EndAfterHour >= 0 {
		args = append(args, query.EndAfterHour)
		innerWhere = fmt.Sprintf("%s AND EXTRACT(HOUR FROM end_time AT TIME ZONE 'EDT') >= $%d", innerWhere, argIndex)
		argIndex++
	}

	titleRank := "1"
	searchRank := "1"

	tsquery := strings.Join(query.TextQueries, " & ")
	tsquery = strings.ReplaceAll(tsquery, "'", "")
	if len(tsquery) > 0 {
		args = append(args, tsquery)
		qArg := fmt.Sprintf("$%d", argIndex)
		argIndex++

		innerFrom = fmt.Sprintf("%s, websearch_to_tsquery('english', %s) q", innerFrom, qArg)
		innerWhere = fmt.Sprintf("%s AND search_key @@ q", innerWhere)
		titleRank = "min(ts_rank(title_tsv, q))"
		searchRank = "min(ts_rank(search_key, q))"
	}

	innerQuery := fmt.Sprintf(`
SELECT
    min(event_id) as event_id,
	min(start_time) as start_time,
	count(active or null) as num_events,
	sum(tickets_available) as tickets_available,
	count(CASE WHEN day_of_week = 3 THEN 1 ELSE NULL END) as wed_events,
	sum(CASE WHEN day_of_week = 3 THEN max_players ELSE 0 END) as wed_total_tickets,
	sum(CASE WHEN day_of_week = 3 THEN tickets_available ELSE 0 END) as wed_tickets,
	count(CASE WHEN day_of_week = 4 THEN 1 ELSE NULL END) as thu_events,
	sum(CASE WHEN day_of_week = 4 THEN max_players ELSE 0 END) as thu_total_tickets,
	sum(CASE WHEN day_of_week = 4 THEN tickets_available ELSE 0 END) as thu_tickets,
	count(CASE WHEN day_of_week = 5 THEN 1 ELSE NULL END) as fri_events,
	sum(CASE WHEN day_of_week = 5 THEN max_players ELSE 0 END) as fri_total_tickets,
	sum(CASE WHEN day_of_week = 5 THEN tickets_available ELSE 0 END) as fri_tickets,
	count(CASE WHEN day_of_week = 6 THEN 1 ELSE NULL END) as sat_events,
	sum(CASE WHEN day_of_week = 6 THEN max_players ELSE 0 END) as sat_total_tickets,
	sum(CASE WHEN day_of_week = 6 THEN tickets_available ELSE 0 END) as sat_tickets,
	count(CASE WHEN day_of_week = 0 THEN 1 ELSE NULL END) as sun_events,
	sum(CASE WHEN day_of_week = 0 THEN max_players ELSE 0 END) as sun_total_tickets,
	sum(CASE WHEN day_of_week = 0 THEN tickets_available ELSE 0 END) as sun_tickets,
    %s as title_rank,
    %s as search_rank
FROM %s
WHERE %s
GROUP BY cluster_id
`, titleRank, searchRank, innerFrom, innerWhere)

	// Default to true so we don't filter anything out
	// if no days were requested
	dayPart := "true"
	if len(query.DaysOfWeek) > 0 {
		var days []string
		for d := range query.DaysOfWeek {
			if query.DaysOfWeek[d] {
				days = append(days, fmt.Sprintf("c.%v_tickets > 0", d))
			}
		}
		dayPart = strings.Join(days, " OR ")
	}
	fullWhere := fmt.Sprintf("e.year = %s AND (%s)", yearArg, dayPart)

	if query.OrgId > 0 {
		args = append(args, query.OrgId)
		fullWhere = fmt.Sprintf("(%s) AND o.id = $%d", fullWhere, argIndex)
	}

	// #nosec G201
	fullQuery := fmt.Sprintf(`
SELECT  distinct
		e.event_id,
		e.title,
		e.short_description,
		e.short_category,
		e.game_system,
		e.org_group,
		o.id AS org_id,
		c.num_events,
		c.tickets_available,
		c.wed_events,
		c.wed_total_tickets,
		c.wed_tickets,
		c.thu_events,
		c.thu_total_tickets,
		c.thu_tickets,
		c.fri_events,
		c.fri_total_tickets,
		c.fri_tickets,
		c.sat_events,
		c.sat_total_tickets,
		c.sat_tickets,
		c.sun_events,
		c.sun_total_tickets,
		c.sun_tickets,
		c.title_rank as title_rank,
		c.search_rank as search_rank		
FROM events e JOIN (%s) AS c ON e.event_id = c.event_id
    JOIN (
        SELECT lower(alias) as alias, MAX(id) as id
        FROM orgs
        GROUP BY lower(alias)
    ) o ON o.alias = lower(e.org_group)
WHERE %s
ORDER BY c.title_rank desc, c.search_rank desc, c.tickets_available desc
`, innerQuery, fullWhere)

	loadedEvents := make([]*EventGroup, 0)
	rows, err := db.Query(fullQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Load all the events
	for rows.Next() {
		group, err := rowToGroup(rows)
		if err != nil {
			return nil, err
		}

		loadedEvents = append(loadedEvents, group)
	}

	log.Printf("Loaded %v events: ", len(loadedEvents))
	return loadedEvents, nil
}

type PersistedEventInfo struct {
	LastModified     time.Time
	TicketsAvailable int
}

func loadEventIds(tx *sql.Tx, year int) (map[string]PersistedEventInfo, map[string]PersistedEventInfo, error) {
	// load all events: ids + last update time + tickets available
	rows, err := tx.Query(`
SELECT event_id, active, last_modified, tickets_available
FROM events
WHERE year=$1`, year)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	activeEvents := make(map[string]PersistedEventInfo)
	inactiveEvents := make(map[string]PersistedEventInfo)

	for rows.Next() {
		var id string
		var active bool
		var info PersistedEventInfo

		err := rows.Scan(&id, &active, &info.LastModified, &info.TicketsAvailable)
		if err != nil {
			return nil, nil, err
		}

		if active {
			activeEvents[id] = info
		} else {
			inactiveEvents[id] = info
		}
	}

	return activeEvents, inactiveEvents, nil
}

func bulkDelete(tx *sql.Tx, deletedEvents []string) error {
	// Deletes aren't true deletes, we mark them as inactive
	if len(deletedEvents) == 0 {
		return nil
	}
	_, err := tx.Exec("UPDATE events SET active = FALSE WHERE event_id = ANY($1)", pq.Array(deletedEvents))
	if err != nil {
		log.Printf("Error on bulk delete: %v", err)
		return err
	}
	return nil
}

type UpdateStats struct {
	Seen      int
	Inserted  int
	Updated   int
	Deleted   int
	Unchanged int
}

func BulkUpdateEvents(tx *sql.Tx, parsedEvents []*events.GenconEvent) (UpdateStats, error) {
	var stats UpdateStats
	if len(parsedEvents) == 0 {
		return stats, nil
	}
	stats.Seen = len(parsedEvents)
	year := parsedEvents[0].Year
	activeEvents, inactiveEvents, err := loadEventIds(tx, year)
	if err != nil {
		return stats, err
	}
	persistedEvents := make(map[string]PersistedEventInfo, len(activeEvents)+len(inactiveEvents))
	for id, info := range activeEvents {
		persistedEvents[id] = info
	}
	for id, info := range inactiveEvents {
		persistedEvents[id] = info
	}

	log.Printf("Loaded %d Rows\n", len(persistedEvents))

	var newEvents []*events.GenconEvent
	var updatedEvents []*events.GenconEvent

	for _, parsedEvent := range parsedEvents {
		if info, found := persistedEvents[parsedEvent.EventId]; found {
			_, isActive := activeEvents[parsedEvent.EventId]
			if !isActive || info.LastModified.Truncate(time.Second).Before(parsedEvent.LastModified.Truncate(time.Second)) || info.TicketsAvailable != parsedEvent.TicketsAvailable {
				updatedEvents = append(updatedEvents, parsedEvent)
			} else {
				stats.Unchanged++
			}
			delete(activeEvents, parsedEvent.EventId)
		} else {
			newEvents = append(newEvents, parsedEvent)
		}
	}

	// Any remaining active events should be deleted
	deletedEvents := make([]string, 0, len(activeEvents))
	for event := range activeEvents {
		deletedEvents = append(deletedEvents, event)
	}

	stats.Inserted = len(newEvents)
	stats.Updated = len(updatedEvents)
	stats.Deleted = len(deletedEvents)

	log.Printf("Inserting %d events\n", stats.Inserted)
	log.Printf("Updating %d events\n", stats.Updated)
	log.Printf("Deleting %d events\n", stats.Deleted)
	log.Printf("Unchanged %d events\n", stats.Unchanged)

	err = bulkInsert(tx, newEvents)
	if err != nil {
		return stats, err
	}
	err = bulkUpdate(tx, updatedEvents)
	if err != nil {
		return stats, err
	}
	err = bulkDelete(tx, deletedEvents)
	if err != nil {
		return stats, err
	}

	// Set-based organizer alias merging
	_, err = tx.Exec(`
		INSERT INTO orgs (alias)
		SELECT DISTINCT org_group FROM events 
		WHERE org_group IS NOT NULL AND org_group != ''
		  AND NOT EXISTS (
			  SELECT 1 FROM orgs WHERE orgs.alias = events.org_group
		  )
	`)
	if err != nil {
		return stats, err
	}

	_, err = tx.Exec(`
		UPDATE orgs o
		SET id = sub.min_id
		FROM (
			SELECT o1.alias, MIN(o2.id) as min_id
			FROM orgs o1
			JOIN orgs o2 ON TRANSLATE(LOWER(o1.alias), '''.",!:; ', '') = TRANSLATE(LOWER(o2.alias), '''.",!:; ', '')
			GROUP BY o1.alias
		) sub
		WHERE o.alias = sub.alias AND o.id != sub.min_id
	`)
	return stats, err
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
		&event.ShortCategory,
		&event.IsStarred,
		&event.OrgId)

	event.StartTime = event.StartTime.In(INDIANAPOLIS)
	event.EndTime = event.EndTime.In(INDIANAPOLIS)
	return &event, err
}

func bulkUpdate(tx *sql.Tx, updatedRows []*events.GenconEvent) error {
	eventFields := eventFields()
	numEventFields := len(eventFields)

	for _, row := range updatedRows {
		updatedFields := fmt.Sprintf(
			"(%s) = %s",
			strings.Join(eventFields, ", "),
			fmt.Sprintf(
				"($%d"+strings.Repeat(", $%d", numEventFields-1)+")",
				rangeSlice(1, numEventFields)...))
		// #nosec G201
		updateStatement := fmt.Sprintf(
			"UPDATE events SET %s WHERE event_id=$%d",
			updatedFields,
			numEventFields+1)

		valueArgs := eventToDbFields(row)
		valueArgs = append(valueArgs, row.EventId)
		_, err := tx.Exec(updateStatement, valueArgs...)

		if err != nil {
			log.Printf("Error on updating event: %v %v", row, err.(*pq.Error))
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
		// #nosec G201
		insertStatement := fmt.Sprintf(
			"INSERT INTO events (%s) VALUES %s",
			strings.Join(eventFields, ","),
			strings.Join(valueStrings, ","))
		_, err := tx.Exec(insertStatement, valueArgs...)

		if err != nil {
			log.Printf("Error on processing event: %v %v", batch, err.(*pq.Error))
			return err
		}
	}

	return nil
}

func GetLastUpdate(db *sql.DB) (time.Time, error) {
	var lastUpdate time.Time
	err := db.QueryRow("SELECT timestamp FROM update_log WHERE success = true ORDER BY timestamp DESC LIMIT 1").Scan(&lastUpdate)
	if err == sql.ErrNoRows {
		err = db.QueryRow("SELECT max(last_modified) FROM events").Scan(&lastUpdate)
	}
	return lastUpdate, err
}

func LogUpdate(db *sql.DB, stats UpdateStats, success bool, errorMsg string) error {
	_, err := db.Exec(`
		INSERT INTO update_log (
			success, events_seen, events_inserted, events_updated, events_deleted, events_unchanged, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		success, stats.Seen, stats.Inserted, stats.Updated, stats.Deleted, stats.Unchanged, errorMsg)
	return err
}
