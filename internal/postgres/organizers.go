package postgres

import (
	"database/sql"
	"github.com/lib/pq"
	log "log"
	"sort"
)

type Organizer struct {
	Id int64
	Aliases []string
	NumEvents int64
}

func MergeOrgs(db *sql.DB, orgs []int64) {
	if len(orgs) < 2 {
		return
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
	}
}

func LoadAllOrgs(db *sql.DB) ([]*Organizer, error) {
	rows, err := db.Query(`
SELECT o.id, array_agg(distinct e.org_group), count(distinct e.event_id)
FROM orgs o LEFT JOIN events e ON (lower(o.alias) = lower(e.org_group))
GROUP BY 1
`)
	if err != nil {
		return nil, err
	}

	orgs := make([]*Organizer, 0, 0)
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