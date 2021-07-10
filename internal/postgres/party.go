package postgres

import (
	"database/sql"
	"log"
)

// A party is a group of users playing together in a given year.
type Party struct {
	Id      int64
	Name    string
	Year    int64
	Members []User
}

func NewParty(db *sql.DB, name string, year int64) (*Party, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	// Cleanup transaction!
	defer func() {
		var txErr error
		if err != nil {
			txErr = tx.Rollback()
		} else {
			txErr = tx.Commit()
		}
		if txErr != nil {
			log.Printf("Error while resolving transaction: %v", txErr)
		}
	}()

	//tx.Exec()

	return nil, nil
}
