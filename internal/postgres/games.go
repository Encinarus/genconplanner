package postgres

import (
	"database/sql"
	"github.com/lib/pq"
	"log"
	"time"
)

type GameFamily struct {
	Name       string
	BggId      int64
	GameIds    []int64
	LastUpdate time.Time
}

func (gf *GameFamily) Upsert(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
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

	_, err = tx.Exec(`
INSERT INTO boardgame_family
    (name, bgg_id, game_ids, last_update)
VALUES 
    ($1, $2, $3, CURRENT_DATE)
ON CONFLICT (bgg_id) 
    DO UPDATE SET name = $1, game_ids = $3, last_update = CURRENT_DATE
`, gf.Name, gf.BggId, pq.Array(gf.GameIds))

	if err != nil {
		return err
	}

	return err
}

func LoadFamilies(db *sql.DB) ([]*GameFamily, error) {
	rows, err := db.Query(`
SELECT 
    name,
	bgg_id,
    game_ids,
    last_update
FROM boardgame_family bg
`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	families := make([]*GameFamily, 0)

	for rows.Next() {
		var gf GameFamily
		var timeHolder pq.NullTime
		err = rows.Scan(
			&gf.Name, &gf.BggId, pq.Array(&gf.GameIds), &timeHolder)
		if err != nil {
			return nil, err
		}
		// We don't check for valid since they'll default to 0 anyway.
		gf.LastUpdate = timeHolder.Time
		families = append(families, &gf)
	}
	return families, nil
}

type Game struct {
	Name          string
	Type          string // Will be game, or expansion
	BggId         int64
	FamilyIds     []int64
	LastUpdate    time.Time
	NumRatings    int64
	AvgRatings    float64
	YearPublished int64
}

func (g *Game) Upsert(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
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

	_, err = tx.Exec(`
INSERT INTO boardgame
    (name, bgg_id, family_ids, num_ratings, avg_ratings, year_published, type, last_update)
VALUES 
    ($1, $2, $3, $4, $5, $6, $7, CURRENT_DATE)
ON CONFLICT (bgg_id) 
    DO UPDATE SET name = $1, family_ids = $3, num_ratings = $4, avg_ratings = $5, year_published = $6, type = $7, last_update = CURRENT_DATE
`, g.Name, g.BggId, pq.Array(g.FamilyIds), g.NumRatings, g.AvgRatings, g.YearPublished, g.Type)

	if err != nil {
		return err
	}

	return err
}

func LoadGames(db *sql.DB) ([]*Game, error) {
	rows, err := db.Query(`
SELECT 
    name,
	bgg_id, 
    family_ids,
    num_ratings,
    avg_ratings,
    year_published,
    type,
    last_update
FROM boardgame bg
`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	games := make([]*Game, 0)

	for rows.Next() {
		var g Game
		var timeHolder pq.NullTime
		var numRatingHolder sql.NullInt64
		var avgRatingHolder sql.NullFloat64
		var yearPublishedHolder sql.NullInt64
		var typeHolder sql.NullString
		err = rows.Scan(
			&g.Name, &g.BggId, pq.Array(&g.FamilyIds),
			&numRatingHolder, &avgRatingHolder, &yearPublishedHolder,
			&typeHolder, &timeHolder)
		if err != nil {
			return nil, err
		}
		// We don't check for valid since they'll default to 0 anyway.
		g.NumRatings = numRatingHolder.Int64
		g.AvgRatings = avgRatingHolder.Float64
		g.YearPublished = yearPublishedHolder.Int64

		if typeHolder.Valid {
			g.Type = typeHolder.String
		}
		games = append(games, &g)
	}
	return games, nil
}
