package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Encinarus/genconplanner/internal/bgg"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"log"
)

func main() {
	flag.Parse()

	db, err := postgres.OpenDb()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	api := bgg.NewBggApi()

	families := make(map[int]*postgres.GameFamily)
	games := make(map[int]*postgres.Game)

	db_games, err := postgres.LoadGames(db)
	if err != nil {
		log.Fatal(err)
	}

	for _, g := range db_games {
		games[g.BggId] = g
	}

	nextFamilies := make([]int, 0, 0)
	nextFamilies = append(nextFamilies, 65191)
	for len(nextFamilies) > 0 {
		log.Printf("Next batch size: %v", len(nextFamilies))
		log.Printf("Processed %v families, %v games", len(families), len(games))

		nextGames := make([]int, 0, 0)
		for _, id := range nextFamilies {
			fam, err := api.GetFamily(ctx, id)
			if err != nil {
				log.Printf("Issue getting family %v", err)
				continue
			}

			families[id] = &postgres.GameFamily{
				Name:  fam.Item.Name.Value,
				BggId: fam.Item.ID,
			}
			for _, related := range fam.Item.Link {
				if _, found := games[related.ID]; !found {
					nextGames = append(nextGames, related.ID)
				}
			}
		}

		nextFamilies = make([]int, 0, 0)
		for _, id := range nextGames {
			game, err := api.GetGame(ctx, id)
			if err != nil {
				log.Printf("Issue getting game %v", err)
				continue
			}
			for _, related := range game.Item.Link {
				if related.Type != "boardgamefamily" {
					continue
				}

				if _, found := families[related.ID]; !found {
					nextFamilies = append(nextFamilies, related.ID)
				}
			}

			// Default to 0 just in case none of them are primary
			name := game.Item.Name[0].Value
			for _, n := range game.Item.Name {
				if n.Type == "primary" {
					name = n.Value
					break
				}
			}
			g := &postgres.Game{
				Name:      name,
				BggId:     game.Item.ID,
				FamilyIds: nil,
			}
			if err = g.Upsert(db); err != nil {
				log.Printf("Issue storing game %v", err)
				continue
			}
			games[id] = g

		}
	}
	text, _ := json.MarshalIndent(families, "", "  ")
	fmt.Printf("Family: %v", string(text))
}
