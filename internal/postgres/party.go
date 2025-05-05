package postgres

import (
	"database/sql"

	"github.com/lib/pq"
)

// A party is a group of users playing together in a given year.
type Party struct {
	Id      int64
	Name    string
	Year    int64
	Members []*User
}

func LoadParties(db *sql.DB, currentUser *User) ([]*Party, error) {
	// Load all partiesById the current user is in
	rows, err := db.Query(
		`
SELECT p.party_id,
       p.name,
       p.year
FROM parties p
    JOIN party_members pm ON p.party_id = pm.party_id
WHERE pm.email = $1
`, currentUser.Email)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	partyIds := make([]int64, 0)
	partiesById := make(map[int64]*Party)
	for rows.Next() {
		var p Party
		err = rows.Scan(&p.Id, &p.Name, &p.Year)
		if err != nil {
			return nil, err
		}
		partiesById[p.Id] = &p
		partyIds = append(partyIds, p.Id)
	}
	rows, err = db.Query(
		`
select u.email, CASE
                    WHEN length(u.display_name) > 0
                        THEN u.display_name
                    ELSE split_part(u.email, '@', 1)
    END, ARRAY_AGG(pm.party_id)
FROM party_members pm join users u on u.email = pm.email
WHERE pm.party_id = ANY($1)
GROUP BY u.email, u.display_name
`, pq.Array(partyIds))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var u User
		userParties := make([]int64, 0)
		err = rows.Scan(&u.Email, &u.DisplayName, pq.Array(&userParties))
		if err != nil {
			return nil, err
		}
		for _, partyId := range userParties {
			partiesById[partyId].Members = append(partiesById[partyId].Members, &u)
		}
	}

	parties := make([]*Party, 0)
	for _, party := range partiesById {
		parties = append(parties, party)
	}

	return parties, nil
}

func NewParty(db *sql.DB, name string, year int64, founderEmail string) (*Party, error) {
	founder, err := LoadOrCreateUser(db, founderEmail)

	if err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	defer func() { CleanupTransaction(err, tx) }()

	var partyId int64
	err = tx.QueryRow(`
INSERT INTO parties(name, year) VALUES ($1, $2) RETURNING party_id`, name, year).Scan(&partyId)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
INSERT INTO party_members (party_id, email) VALUES ($1, $2)`, partyId, founder.Email)
	if err != nil {
		return nil, err
	}

	return &Party{
		Id:      partyId,
		Name:    name,
		Year:    year,
		Members: []*User{founder},
	}, nil
}
