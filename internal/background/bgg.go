package background

import (
	"context"
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/bgg"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"log"
	"time"
)

func UpdateGamesFromBGG(db *sql.DB) {
	ctx := context.Background()
	api := bgg.NewBggApi()

	families := make(map[int64]*postgres.GameFamily)
	games := make(map[int64]*postgres.Game)

	dbGames, err := postgres.LoadGames(db)
	if err != nil {
		log.Printf("Unable to load games, continuing %v", err)
	}

	nextFamilies := map[int64]bool{
		65191: true, 27646: true, 71181: true, 66772: true, 6258: true, 65328: true, 41489: true,
	}

	addFamilies := func(familyMap map[int64]bool, newFamilies []int64) {
		for _, familyId := range newFamilies {
			if _, found := families[familyId]; !found {
				familyMap[familyId] = true
			}
		}
	}
	for _, g := range dbGames {
		games[g.BggId] = g
		addFamilies(nextFamilies, g.FamilyIds)
	}

	// If we haven't updated in 2 weeks, update now
	updateCutoff := time.Now().Add(-time.Hour * 24 * 14)

	log.Printf("Update cutoff: %v", updateCutoff)

	for len(nextFamilies) > 0 {
		log.Printf("Next batch size: %v", len(nextFamilies))
		log.Printf("Processed %v families, %v games", len(families), len(games))

		nextGames := make([]int64, 0, 0)
		for id := range nextFamilies {
			fam, err := api.GetFamily(ctx, id)
			if err != nil {
				log.Printf("Issue getting family: %v", err)
				continue
			}

			families[id] = &postgres.GameFamily{
				Name:  fam.Item.Name.Value,
				BggId: fam.Item.ID,
			}
			for _, related := range fam.Item.Link {
				nextGames = append(nextGames, related.ID)
			}
		}

		nextFamilies = make(map[int64]bool)
		for _, id := range nextGames {
			dbGame, found := games[id]
			if found && dbGame.LastUpdate.After(updateCutoff) {
				// We still want this for identifying families to load
				addFamilies(nextFamilies, dbGame.FamilyIds)
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

			addFamilies(nextFamilies, familyIds)

			// Default to 0 just in case none of them are primary
			name := apiGame.Item.Name[0].Value
			for _, n := range apiGame.Item.Name {
				if n.Type == "primary" {
					name = n.Value
					break
				}
			}

			g := &postgres.Game{
				Name:       name,
				BggId:      apiGame.Item.ID,
				FamilyIds:  familyIds,
				LastUpdate: time.Now(),
				NumRatings: apiGame.Item.Statistics.Ratings.Usersrated.Value,
				AvgRatings: apiGame.Item.Statistics.Ratings.Average.Value,
			}
			if err = g.Upsert(db); err != nil {
				log.Printf("Issue storing apiGame %v", err)
				continue
			}
			games[id] = g
		}
	}
}
