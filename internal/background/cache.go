package background

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"log"
	"strings"
	"sync"
	"time"
)

type GameCache struct {
	// Name -> games
	games map[string][]*postgres.Game // guarded by mu

	db *sql.DB // threadsafe, not guarded by mutex

	mu sync.Mutex
}

func NewGameCache(db *sql.DB) *GameCache {
	return &GameCache{
		games: make(map[string][]*postgres.Game),
		db:    db,
	}
}

func (gc *GameCache) PeriodicallyUpdate() {
	bgTicker := time.NewTicker(time.Hour)

	go func() {
		for {
			err := gc.UpdateCache()
			if err != nil {
				log.Printf("Error updating cache: %v", err)
			}

			select {
			case <-bgTicker.C:
			}
		}
	}()

}

func (gc *GameCache) UpdateCache() error {
	dbGames, err := postgres.LoadGames(gc.db)
	if err != nil {
		return err
	}

	newGames := make(map[string][]*postgres.Game)

	for _, g := range dbGames {
		newGames[strings.TrimSpace(strings.ToLower(g.Name))] = append(newGames[g.Name], g)
	}

	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.games = newGames

	return nil
}

func (gc *GameCache) FindGame(name string) []*postgres.Game {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	return gc.games[strings.TrimSpace(strings.ToLower(name))]
}
