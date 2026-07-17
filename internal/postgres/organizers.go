package postgres

import (
	"database/sql"
	log "log"
	"sort"

	"github.com/lib/pq"
)

type Organizer struct {
	Id        int64
	Aliases   []string
	NumEvents int64
}

func MergeOrgs(db *sql.DB, orgs []int64) error {
	if len(orgs) < 2 {
		return nil
	}
	// The lowest numbered org will be the winner
	sort.Slice(orgs, func(i, j int) bool {
		return orgs[i] < orgs[j]
	})
	smallest := orgs[0]
	orgs = orgs[1:]

	log.Printf("Merging orgs, smallest %v, merges: %v", smallest, orgs)

	_, err := db.Exec(`UPDATE orgs SET id = $1 WHERE id = ANY ($2)`,
		smallest, pq.Array(orgs))
	if err != nil {
		log.Printf("Error when updating orgs: %v", err)
		return err
	}
	return nil
}

func LoadAllOrgs(db *sql.DB) ([]*Organizer, error) {
	rows, err := db.Query(`
SELECT o.id, coalesce(array_agg(distinct e.org_group) filter (where e.org_group is not null), '{}'), count(distinct e.event_id)
FROM orgs o LEFT JOIN events e ON (lower(o.alias) = lower(e.org_group))
GROUP BY 1
`)
	if err != nil {
		return nil, err
	}

	orgs := make([]*Organizer, 0)
	defer rows.Close()
	for rows.Next() {
		var org Organizer
		err = rows.Scan(&org.Id, pq.Array(&org.Aliases), &org.NumEvents)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, &org)
	}
	return orgs, nil
}

type EventOrgMetadata struct {
	OrgGroup string
	Title    string
	Year     int
	GmNames  string
	Email    string
	Website  string
}

func LoadEventOrgMetadata(db *sql.DB) ([]EventOrgMetadata, error) {
	rows, err := db.Query(`
SELECT coalesce(org_group, ''), coalesce(title, ''), coalesce(year, 0), coalesce(gm_names, ''), coalesce(email, ''), coalesce(website, '')
FROM events
WHERE org_group IS NOT NULL AND org_group <> ''
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []EventOrgMetadata
	for rows.Next() {
		var m EventOrgMetadata
		err = rows.Scan(&m.OrgGroup, &m.Title, &m.Year, &m.GmNames, &m.Email, &m.Website)
		if err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, nil
}

