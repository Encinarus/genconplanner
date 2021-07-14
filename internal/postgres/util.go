package postgres

import "database/sql"

// Use as defer func() { CleanupTransaction(err, tx) }()
// Need to do it in an anonymous function to avoid binding err and tx
func CleanupTransaction(err error, tx *sql.Tx) {
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
}
