package background

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Encinarus/genconplanner/internal/bgg"
	"github.com/Encinarus/genconplanner/internal/postgres"
)

type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
func (RealClock) Sleep(d time.Duration) { time.Sleep(d) }

func addIdsToBacklog(backlog map[int64]bool, newIds []int64) {
	for _, id := range newIds {
		if _, found := backlog[id]; !found {
			backlog[id] = true
		}
	}
}

func RefreshGame(ctx context.Context, apiGame *bgg.GameItem,
	familyBacklog map[int64]bool, db *sql.DB, clock Clock) (*postgres.Game, error) {

	var familyIds []int64

	for _, related := range apiGame.Link {
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
	name := apiGame.Name[0].Value
	for _, n := range apiGame.Name {
		if n.Type == "primary" {
			name = n.Value
			break
		}
	}

	g := &postgres.Game{
		Name:          name,
		BggId:         apiGame.ID,
		FamilyIds:     familyIds,
		LastUpdate:    clock.Now(),
		NumRatings:    apiGame.Statistics.Ratings.NumRatings.Value,
		AvgRatings:    apiGame.Statistics.Ratings.Average.Value,
		YearPublished: apiGame.YearPublished.Value,
		Type:          apiGame.Type,
	}
	if err := g.Upsert(db); err != nil {
		log.Printf("Issue storing apiGame %v", err)
		return nil, err
	}
	return g, nil
}

func UpdateGamesFromBGG(ctx context.Context, db *sql.DB, api bgg.BGGClient, clock Clock) {
	if api == nil {
		log.Println("BGG API client not set, skipping BGG update.")
		return
	}

	// Initial seed with kickstarter, this is a big category, good for branching out everywhere :)
	familyBacklog := map[int64]bool{
		8374: true,
	}

	gameBacklog := make(map[int64]bool)
	for _, id := range SeedBGGGameIds() {
		gameBacklog[id] = true
	}

	families := make(map[int64]*postgres.GameFamily)
	games := make(map[int64]*postgres.Game)

	log.Printf("Beginning update of games from BGG, initial game backlog: %v", len(gameBacklog))

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
	familyUpdateLimit := clock.Now().Add(-time.Hour * 24 * 4)
	// If we haven't updated in 4 weeks, update now. Once we know about a game, it's probably fairly stable.
	// With a rate limit of one call per 5 seconds, we can process ~438k games.
	gameUpdateLimit := clock.Now().Add(-time.Hour * 24 * 28)

	for len(familyBacklog) > 0 || len(gameBacklog) > 0 {
		log.Printf("Processing backlog")
		log.Printf("  Family backlog: %v", len(familyBacklog))
		log.Printf("  Game backlog: %v", len(gameBacklog))
		log.Printf("  Processed %v families, %v games", len(families), len(games))

		processedGames := 0
		processedFamilies := 0

		// Prioritize unknown games.
		var unknownIds []int64
		for id := range gameBacklog {
			if _, found := games[id]; !found {
				unknownIds = append(unknownIds, id)
			}
		}

		batchSize := 20
		for i := 0; i < len(unknownIds); i += batchSize {
			end := i + batchSize
			if end > len(unknownIds) {
				end = len(unknownIds)
			}
			batch := unknownIds[i:end]

			apiGames, err := api.GetGames(ctx, batch)
			if err != nil {
				log.Printf("Issue getting apiGames %v", err)
				continue
			}
			logDetails := ""
			for _, apiGame := range apiGames {
				processedGames++
				g, err := RefreshGame(ctx, apiGame, familyBacklog, db, clock)
				if err == nil {
					games[g.BggId] = g
					if logDetails != "" {
						logDetails += ", "
					}
					logDetails += fmt.Sprintf("%d:%s", g.BggId, g.Name)
				}
			}
			log.Printf("processed %d games... %s", len(apiGames), logDetails)
		}

		var oldIds []int64
		for id := range gameBacklog {
			dbGame, found := games[id]
			if !found {
				continue
			} else if dbGame.LastUpdate.After(gameUpdateLimit) {
				// We still want this for identifying families to load
				addIdsToBacklog(familyBacklog, dbGame.FamilyIds)
				continue
			}
			oldIds = append(oldIds, id)
		}

		for i := 0; i < len(oldIds); i += batchSize {
			end := i + batchSize
			if end > len(oldIds) {
				end = len(oldIds)
			}
			batch := oldIds[i:end]

			apiGames, err := api.GetGames(ctx, batch)
			if err != nil {
				log.Printf("Issue getting apiGames %v", err)
				continue
			}
			logDetails := ""
			for _, apiGame := range apiGames {
				processedGames++
				g, err := RefreshGame(ctx, apiGame, familyBacklog, db, clock)
				if err == nil {
					games[g.BggId] = g
					if logDetails != "" {
						logDetails += ", "
					}
					logDetails += fmt.Sprintf("%d:%s", g.BggId, g.Name)
				}
			}
			log.Printf("processed %d games... %s", len(apiGames), logDetails)
		}

		gameBacklog = make(map[int64]bool)
		var familiesToProcess []int64
		for id := range familyBacklog {
			dbFamily, found := families[id]
			if found && dbFamily.LastUpdate.After(familyUpdateLimit) {
				addIdsToBacklog(gameBacklog, dbFamily.GameIds)
				delete(familyBacklog, id)
				continue
			}
			familiesToProcess = append(familiesToProcess, id)
		}

		for i := 0; i < len(familiesToProcess); i += batchSize {
			end := i + batchSize
			if end > len(familiesToProcess) {
				end = len(familiesToProcess)
			}
			batch := familiesToProcess[i:end]

			apiFamilies, err := api.GetFamilies(ctx, batch)
			if err != nil {
				log.Printf("Issue getting families: %v", err)
				continue
			}

			logDetails := ""
			for _, bggFamily := range apiFamilies {
				processedFamilies++
				gameIds := make([]int64, 0, len(bggFamily.Link))
				for _, related := range bggFamily.Link {
					gameIds = append(gameIds, related.ID)
				}
				addIdsToBacklog(gameBacklog, gameIds)

				dbFamily := &postgres.GameFamily{
					Name:       bggFamily.Name.Value,
					BggId:      bggFamily.ID,
					GameIds:    gameIds,
					LastUpdate: clock.Now(),
				}
				families[bggFamily.ID] = dbFamily
				err = families[bggFamily.ID].Upsert(db)
				if err != nil {
					log.Printf("Issue saving family: %v", err)
					continue
				}
				if logDetails != "" {
					logDetails += ", "
				}
				logDetails += fmt.Sprintf("%d:%s", bggFamily.ID, bggFamily.Name.Value)
			}
			log.Printf("processed %d families... %s", len(apiFamilies), logDetails)
		}

		// We're done! We don't know about anything else to dig into
		if processedFamilies == 0 && processedGames == 0 {
			log.Printf("No updates needed, finishing BGG update pass")
			return
		}
	}
}
