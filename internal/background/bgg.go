package background

import (
	"context"
	"database/sql"
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

func UpdateGamesFromBGG(db *sql.DB) {
	ctx := context.Background()
	api := bgg.NewBggApi()

	// Initial seed with kickstarter, this is a big category, good for branching out everywhere :)
	familyBacklog := map[int64]bool{
		8374: true,
	}
	gameBacklog := make(map[int64]bool)

	families := make(map[int64]*postgres.GameFamily)
	games := make(map[int64]*postgres.Game)

	dbFamilies, err := postgres.LoadFamilies(db)
	if err != nil {
		log.Printf("Unable to load game families, continuing %v", err)
	}
	for _, gf := range dbFamilies {
		families[gf.BggId] = gf
		addIdsToBacklog(gameBacklog, gf.GameIds)
	}

	dbGames, err := postgres.LoadGames(db)
	if err != nil {
		log.Printf("Unable to load games, continuing %v", err)
	}
	for _, g := range dbGames {
		games[g.BggId] = g
		addIdsToBacklog(familyBacklog, g.FamilyIds)
	}

	// If we haven't updated in 4 days, update now
	familyUpdateLimit := time.Now().Add(-time.Hour * 24 * 4)
	// If we haven't updated in 2 weeks, update now
	gameUpdateLimit := time.Now().Add(-time.Hour * 24 * 14)

	for len(familyBacklog) > 0 || len(gameBacklog) > 0 {
		log.Printf("Family backlog: %v", len(familyBacklog))
		log.Printf("Game backlog: %v", len(gameBacklog))
		log.Printf("Processed %v families, %v games", len(families), len(games))

		for id := range familyBacklog {
			dbFamily, found := families[id]
			if found && dbFamily.LastUpdate.After(familyUpdateLimit) {
				addIdsToBacklog(gameBacklog, dbFamily.GameIds)
				continue
			}

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

		familyBacklog = make(map[int64]bool)
		for id := range gameBacklog {
			dbGame, found := games[id]
			if found && dbGame.LastUpdate.After(gameUpdateLimit) {
				// We still want this for identifying families to load
				addIdsToBacklog(familyBacklog, dbGame.FamilyIds)
				continue
			}

			apiGame, err := api.GetGame(ctx, id)
			if err != nil {
				log.Printf("Issue getting apiGame %v", err)
				continue
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
					continue
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
				continue
			}
			games[id] = g
		}
	}
}
