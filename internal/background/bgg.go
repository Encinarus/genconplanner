package background

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Encinarus/genconplanner/internal/bgg"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"log"
	"time"
)

func addIdsToBacklog(backlog map[int64]bool, newIds []int64) {
	for _, id := range newIds {
		if _, found := backlog[id]; !found {
			backlog[id] = true
		}
	}
}

func RefreshGame(ctx context.Context, bggId int64,
	familyBacklog map[int64]bool, db *sql.DB, api *bgg.BggApi) (*postgres.Game, error) {

	apiGame, err := api.GetGame(ctx, bggId)
	if err != nil {
		log.Printf("Issue getting apiGame %v", err)
		return nil, err
	}
	var familyIds []int64

	for _, related := range apiGame.Item.Link {
		// Other types exist (below), unfortunately we can't query for them. If BGG adds
		// support for pulling these down, we can expand how we branch out and discover
		// games.
		//		boardgamecategory
		//		boardgamemechanic
		//		boardgamedesigner
		//		boardgameartist
		//		boardgamepublisher
		if related.Type != "boardgamefamily" {
			return nil, errors.New("Not a board game")
		}
		familyIds = append(familyIds, related.ID)
	}

	addIdsToBacklog(familyBacklog, familyIds)

	// Default to 0 just in case none of them are primary
	name := apiGame.Item.Name[0].Value
	for _, n := range apiGame.Item.Name {
		if n.Type == "primary" {
			name = n.Value
			break
		}
	}

	g := &postgres.Game{
		Name:          name,
		BggId:         apiGame.Item.ID,
		FamilyIds:     familyIds,
		LastUpdate:    time.Now(),
		NumRatings:    apiGame.Item.Statistics.Ratings.NumRatings.Value,
		AvgRatings:    apiGame.Item.Statistics.Ratings.Average.Value,
		YearPublished: apiGame.Item.YearPublished.Value,
		Type:          apiGame.Item.Type,
	}
	if err = g.Upsert(db); err != nil {
		log.Printf("Issue storing apiGame %v", err)
		return nil, err
	}
	return g, nil
}

func UpdateGamesFromBGG(db *sql.DB) {
	ctx := context.Background()
	api := bgg.NewBggApi()

	// Initial seed with kickstarter, this is a big category, good for branching out everywhere :)
	familyBacklog := map[int64]bool{
		8374: true,
	}
	gamesToAddToBacklog := []int64{
		368966, 338067, 350184, 366013, 295770, 322289, 281526, 321608,
		332772, 374173, 316624, 242705, 324090, 258779, 281549, 295895,
		368058, 371972, 311988, 267609, 336986, 363252, 358026, 287673,
		334782, 317119, 325494, 305096, 227935, 350184, 336986, 256680,
		310873, 237179, 332772, 310100, 315610, 276182, 332686, 240980,
		300217, 331106, 311988, 254127, 328866, 329551, 322289, 351913,
		321608, 258779, 337627, 356033, 219650, 361545, 194517, 284189,
		155250, 362944, 340325, 355093, 359871, 295293, 304051, 317321,
		350316, 266064, 342372, 350205, 332398, 256997, 316624, 273910,
		312859, 322656, 322524, 281549, 348450, 295374, 329500, 374173,
		363369, 354570, 274124, 267609, 345584, 286063, 351817, 281474,
		276086, 365717, 314745, 351526, 324090, 349067, 359438, 329716,
		295895, 356123, 313889, 281647, 271601, 288080, 366161,
	}
	gameBacklog := map[int64]bool{}
	for _, id := range gamesToAddToBacklog {
		gameBacklog[id] = true
	}

	families := make(map[int64]*postgres.GameFamily)
	games := make(map[int64]*postgres.Game)

	dbGames, err := postgres.LoadGames(db)
	if err != nil {
		log.Printf("Unable to load games, continuing %v", err)
	}
	for _, g := range dbGames {
		games[g.BggId] = g
		addIdsToBacklog(familyBacklog, g.FamilyIds)
	}

	dbFamilies, err := postgres.LoadFamilies(db)
	if err != nil {
		log.Printf("Unable to load game families, continuing %v", err)
	}
	for _, gf := range dbFamilies {
		families[gf.BggId] = gf
		addIdsToBacklog(gameBacklog, gf.GameIds)
	}

	// If we haven't updated in 4 days, update now. This should get us faster discovery of new games.
	familyUpdateLimit := time.Now().Add(-time.Hour * 24 * 4)
	// If we haven't updated in 4 weeks, update now. Once we know about a game, it's probably fairly stable.
	// With a rate limit of one call per 5 seconds, we can process ~438k games.
	gameUpdateLimit := time.Now().Add(-time.Hour * 24 * 28)

	for len(familyBacklog) > 0 || len(gameBacklog) > 0 {
		log.Printf("Processing backlog")
		log.Printf("  Family backlog: %v", len(familyBacklog))
		log.Printf("  Game backlog: %v", len(gameBacklog))
		log.Printf("  Processed %v families, %v games", len(families), len(games))

		processedGames := 0
		processedFamilies := 0

		// Prioritize unknown games.
		for id := range gameBacklog {
			_, found := games[id]
			if found {
				// Will be picked up next loop
				continue
			}
			processedGames++

			_, err := RefreshGame(ctx, id, familyBacklog, db, api)
			if err != nil {
				log.Printf("Issue getting apiGame %v", err)
				continue
			}
		}
		for id := range gameBacklog {
			dbGame, found := games[id]
			if !found {
				// Handled in first loop
				continue
			} else if dbGame.LastUpdate.After(gameUpdateLimit) {
				// We still want this for identifying families to load
				addIdsToBacklog(familyBacklog, dbGame.FamilyIds)
				continue
			}
			processedGames++

			_, err := RefreshGame(ctx, id, familyBacklog, db, api)
			if err != nil {
				log.Printf("Issue getting apiGame %v", err)
				continue
			}
		}

		gameBacklog = make(map[int64]bool)
		for id := range familyBacklog {
			dbFamily, found := families[id]
			if found && dbFamily.LastUpdate.After(familyUpdateLimit) {
				addIdsToBacklog(gameBacklog, dbFamily.GameIds)
				continue
			}
			processedFamilies++

			bggFamily, err := api.GetFamily(ctx, id)
			if err != nil {
				log.Printf("Issue getting family: %v", err)
				continue
			}
			gameIds := make([]int64, 0, 0)
			for _, related := range bggFamily.Item.Link {
				gameIds = append(gameIds, related.ID)
			}

			dbFamily = &postgres.GameFamily{
				Name:       bggFamily.Item.Name.Value,
				BggId:      bggFamily.Item.ID,
				GameIds:    gameIds,
				LastUpdate: time.Now(),
			}
			families[id] = dbFamily
			err = families[id].Upsert(db)
			if err != nil {
				log.Printf("Issue saving family: %v", err)
				continue
			}
		}

		// We're done! We don't know about anything else to dig into
		if processedFamilies == 0 && processedGames == 0 {
			log.Printf("No updates needed, sleeping for four hours")
			time.Sleep(4 * time.Hour)
		}
	}
}
