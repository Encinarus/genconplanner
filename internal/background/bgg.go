package background

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/Encinarus/genconplanner/internal/bgg"
	"github.com/Encinarus/genconplanner/internal/postgres"
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
		// Seeded from bgg hotlist: advanced search, sort by rank,
		// filter to new year.
		96, 420, 535, 24225, 27162, 30370, 31730, 32146, 35815, 92120, 155250, 193322,
		194517, 216497, 219276, 219650, 221318, 227935, 228657, 237179, 240225, 240980,
		242705, 254127, 256680, 256997, 258779, 258855, 264806, 266064, 267609, 271601,
		273910, 274124, 276086, 276182, 276296, 281474, 281526, 281549, 281647, 284189,
		286063, 287673, 288080, 289939, 291572, 295293, 295374, 295770, 295895, 300217,
		304051, 305096, 310100, 310873, 311988, 312859, 313889, 314745, 315610, 316624,
		317119, 317321, 321608, 322289, 322524, 322656, 324090, 325494, 328866, 329500,
		329551, 329716, 331106, 332398, 332686, 332772, 332779, 334049, 334782, 336986,
		337098, 337627, 338067, 340325, 341416, 342372, 343980, 345584, 346643, 347479,
		348450, 349067, 350184, 350205, 350316, 351476, 351526, 351817, 351913, 354570,
		355093, 355483, 355829, 356033, 356123, 356245, 356629, 356952, 358026, 358515,
		359438, 359504, 359871, 359974, 360226, 361545, 361972, 362944, 362976, 363252,
		363369, 363625, 364655, 365104, 365717, 366013, 366089, 366161, 366267, 366278,
		368058, 368305, 368966, 369270, 369880, 371482, 371922, 371947, 371972, 373106,
		374173, 374567, 375365, 375957, 376683, 377470, 378142, 378737, 378758, 379005,
		379078, 379644, 381356, 381676, 381677, 381819, 383086, 383459, 385245, 385610,
		387163, 387201, 387202, 387263, 238327, 367584, 286559, 184771, 367197, 121193,
		381984, 359156, 344050, 385415, 369436, 28, 386990, 353019, 376472, 368103,
		373914, 344341, 200114, 285110, 331228, 381117, 366470, 407812, 193949,
	}
	gameBacklog := map[int64]bool{}
	for _, id := range gamesToAddToBacklog {
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
		// TODO: extract the ids then sort, unknown first.
		for id := range gameBacklog {
			_, found := games[id]
			if found {
				// Will be picked up by the next loop
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
