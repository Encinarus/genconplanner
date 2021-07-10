package background

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"log"
	"sort"
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

func (gc *GameCache) FindGame(name string) *postgres.Game {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	matches := gc.games[strings.TrimSpace(strings.ToLower(name))]
	if len(matches) == 0 {
		return nil
	}

	// There can be multiple matches, like arkham horror from 2005 vs 1987
	// So, lets pick the latest one. We're sorting so that newer/better
	// games are earlier in the slice.
	sort.Slice(matches, func(i, j int) bool {
		first := matches[i]
		second := matches[j]
		// we might not have pulled down year published yet, so go with it _if_ we have it.
		if first.YearPublished != 0 && second.YearPublished != 0 {
			return first.YearPublished > second.YearPublished
		}
		return matches[i].NumRatings > matches[j].NumRatings
	})

	return matches[0]
}
