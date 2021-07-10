package background

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"log"
	"math"
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
		normalizedName := strings.TrimSpace(strings.ToLower(g.Name))
		newGames[normalizedName] = append(newGames[normalizedName], g)
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
	// So, lets pick the better one. We're sorting so that newer/better
	// games are earlier in the slice.
	sort.Slice(matches, func(i, j int) bool {
		first := matches[i]
		second := matches[j]

		// First, we'll try to figure out which has more buzz. Going for which has more ratings per year, within
		// an order of magnitude. New games with a lot of buzz will have few years and relatively many ratings.
		// Older games should have fewer ratings per year on average. If something is super new, it likely won't be
		// established enough yet.
		if first.YearPublished != 0 && second.YearPublished != 0 {
			magnitude := func(num, denom int64) int { return int(math.Log10(float64(num) / math.Max(float64(denom), 1))) }
			firstRatingsPerYear := magnitude(first.NumRatings, int64(time.Now().Year())-first.YearPublished)
			secondRatingsPerYear := magnitude(second.NumRatings, int64(time.Now().Year())-second.YearPublished)

			// If one is clearly hotter than the other, return that one.
			if firstRatingsPerYear != secondRatingsPerYear {
				return firstRatingsPerYear > secondRatingsPerYear
			}
		}

		return first.AvgRatings > second.AvgRatings
	})

	return matches[0]
}
