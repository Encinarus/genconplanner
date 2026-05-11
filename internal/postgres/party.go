package postgres

import (
	"database/sql"

	"github.com/lib/pq"
)

// A party is a group of users playing together in a given year.
type Party struct {
	Id          int64
	Name        string
	Year        int64
	LeaderEmail string
	Members     []*User
}

func LoadParties(db *sql.DB, currentUser *User) ([]*Party, error) {
	// Load all partiesById the current user is in
	rows, err := db.Query(
		`
SELECT p.party_id,
       p.name,
       p.year,
       p.leader_email
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
		var leaderEmail sql.NullString
		err = rows.Scan(&p.Id, &p.Name, &p.Year, &leaderEmail)
		if err != nil {
			return nil, err
		}
		p.LeaderEmail = leaderEmail.String
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
INSERT INTO parties(name, year, leader_email) VALUES ($1, $2, $3) RETURNING party_id`, name, year, founder.Email).Scan(&partyId)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
INSERT INTO party_members (party_id, email) VALUES ($1, $2)`, partyId, founder.Email)
	if err != nil {
		return nil, err
	}

	return &Party{
		Id:          partyId,
		Name:        name,
		Year:        year,
		LeaderEmail: founder.Email,
		Members:     []*User{founder},
	}, nil
}

func LoadParty(db *sql.DB, id int64) (*Party, error) {
	var p Party
	var leaderEmail sql.NullString
	err := db.QueryRow(`
SELECT party_id, name, year, leader_email
FROM parties
WHERE party_id = $1
`, id).Scan(&p.Id, &p.Name, &p.Year, &leaderEmail)
	if err != nil {
		return nil, err
	}
	p.LeaderEmail = leaderEmail.String

	rows, err := db.Query(`
SELECT u.email, 
       CASE WHEN length(u.display_name) > 0
            THEN u.display_name
            ELSE split_part(u.email, '@', 1)
            END
FROM party_members pm 
JOIN users u ON u.email = pm.email
WHERE pm.party_id = $1
`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u User
		err = rows.Scan(&u.Email, &u.DisplayName)
		if err != nil {
			return nil, err
		}
		p.Members = append(p.Members, &u)
	}

	return &p, nil
}

func UpdatePartyLeader(db *sql.DB, id int64, newLeaderEmail string) error {
	_, err := db.Exec(`UPDATE parties SET leader_email = $1 WHERE party_id = $2`, newLeaderEmail, id)
	return err
}

func RenameParty(db *sql.DB, id int64, name string) error {
	_, err := db.Exec(`UPDATE parties SET name = $1 WHERE party_id = $2`, name, id)
	return err
}

func DeleteParty(db *sql.DB, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { CleanupTransaction(err, tx) }()

	_, err = tx.Exec(`DELETE FROM party_members WHERE party_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM parties WHERE party_id = $1`, id)
	return err
}

func RemoveMember(db *sql.DB, partyId int64, email string) error {
	_, err := db.Exec(`DELETE FROM party_members WHERE party_id = $1 AND email = $2`, partyId, email)
	return err
}

func JoinParty(db *sql.DB, partyId int64, email string) error {
	// Ensure user exists
	_, err := LoadOrCreateUser(db, email)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
INSERT INTO party_members (party_id, email)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, partyId, email)
	return err
}
