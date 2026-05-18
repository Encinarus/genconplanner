//go:build integration

package background_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/bgg"
	"github.com/Encinarus/genconplanner/internal/testutil"
)

type MockClock struct {
	mu     sync.Mutex
	cur    time.Time
	sleeps []time.Duration
}

func (m *MockClock) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cur.IsZero() {
		m.cur = time.Now()
	}
	return m.cur
}

func (m *MockClock) Sleep(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sleeps = append(m.sleeps, d)
	m.cur = m.cur.Add(d)
}

type MockBGGClient struct {
	mu          sync.Mutex
	gamesCalls  [][]int64
	familyCalls [][]int64
	gamesResp   []*bgg.GameItem
	familyResp  []*bgg.FamilyItem
	err         error
}

func (m *MockBGGClient) GetGames(ctx context.Context, ids []int64) ([]*bgg.GameItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gamesCalls = append(m.gamesCalls, ids)
	if m.err != nil {
		return nil, m.err
	}
	return m.gamesResp, nil
}

func (m *MockBGGClient) GetFamilies(ctx context.Context, ids []int64) ([]*bgg.FamilyItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.familyCalls = append(m.familyCalls, ids)
	if m.err != nil {
		return nil, m.err
	}
	return m.familyResp, nil
}

func TestUpdateGamesFromBGG_BatchingAndCaching(t *testing.T) {
	db, err := testutil.SetupTestDB(t)
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	clock := &MockClock{}
	client := &MockBGGClient{}

	// Setup a valid mock game response for the first batch
	gameItem := &bgg.GameItem{
		ID:          1, // Matches first ID in SeedBGGGameIds
		Type:        "boardgame",
		Description: "A legendary board game",
	}
	gameItem.Name = append(gameItem.Name, struct {
		Type  string `xml:"type,attr"`
		Value string `xml:"value,attr"`
	}{Type: "primary", Value: "Mock Legendary Game"})
	gameItem.YearPublished.Value = 2026
	gameItem.Statistics.Ratings.NumRatings.Value = 500
	gameItem.Statistics.Ratings.Average.Value = 9.2

	client.gamesResp = []*bgg.GameItem{gameItem}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run UpdateGamesFromBGG
	background.UpdateGamesFromBGG(ctx, db, client, clock)

	client.mu.Lock()
	defer client.mu.Unlock()

	// Verify batching behavior: SeedBGGGameIds has 5000 IDs, batch size is 20 -> 250 batches.
	if len(client.gamesCalls) == 0 {
		t.Fatalf("expected GetGames calls, got 0")
	}

	for i, batch := range client.gamesCalls {
		if len(batch) > 20 {
			t.Errorf("batch %d exceeded max size 20: got %d", i, len(batch))
		}
	}
}
