//go:build integration

package postgres_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Encinarus/genconplanner/internal/api"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/Encinarus/genconplanner/internal/testutil"
)

func setupSeededDB(t *testing.T) *api.PostgresRepository {
	db, err := testutil.SetupTestDB(t)
	if err != nil {
		t.Fatalf("failed to setup test database: %v", err)
	}

	seedFile := filepath.Join("..", "testutil", "seed.sql")
	content, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed file: %v", err)
	}

	if _, err := db.ExecContext(context.Background(), string(content)); err != nil {
		t.Fatalf("failed to execute seed file: %v", err)
	}

	return &api.PostgresRepository{DB: db}
}

func TestSearchEvents_Integration(t *testing.T) {
	repo := setupSeededDB(t)

	// 1. Search by Category BGM
	q := postgres.SearchQuery{
		CategoryShortCode: "BGM",
		Year:              2026,
	}
	groups, err := repo.SearchEvents(q)
	if err != nil {
		t.Fatalf("SearchEvents failed: %v", err)
	}

	// BGM has 3 events in seed.sql: BGM26ND100001 (active), BGM26ND100002 (active), BGM26ND100003 (inactive)
	// They share the same title/org_group/game_system, so they group together.
	// Since BGM26ND100003 is inactive, only the 2 active ones should be counted in the group!
	if len(groups) == 0 {
		t.Fatalf("expected at least 1 event group, got 0")
	}

	foundCatan := false
	for _, g := range groups {
		if g.GameSystem == "Catan" {
			foundCatan = true
			if g.Count != 2 {
				t.Errorf("expected group count 2 (active events), got %d", g.Count)
			}
		}
	}
	if !foundCatan {
		t.Errorf("Catan event group not found in search results")
	}
}

func TestLoadPartyMemberPurchases_Integration(t *testing.T) {
	repo := setupSeededDB(t)

	// In seed.sql, leader@example.com and member2@example.com have tier='purchased' for BGM26ND100002.
	purchases, err := repo.LoadPartyMemberPurchases(101, 2026)
	if err != nil {
		t.Fatalf("LoadPartyMemberPurchases failed: %v", err)
	}

	count, exists := purchases["BGM26ND100002"]
	if !exists {
		t.Fatalf("expected BGM26ND100002 in purchases map")
	}
	if count != 2 {
		t.Errorf("expected 2 purchases for BGM26ND100002, got %d", count)
	}
}

func TestLoadStarredEventClusters_Integration(t *testing.T) {
	repo := setupSeededDB(t)

	// Load starred events for leader@example.com
	starredEvents, err := repo.LoadStarredEvents("leader@example.com", 2026)
	if err != nil {
		t.Fatalf("LoadStarredEvents failed: %v", err)
	}

	clusters, err := repo.LoadStarredEventClusters("leader@example.com", 2026, starredEvents)
	if err != nil {
		t.Fatalf("LoadStarredEventClusters failed: %v", err)
	}

	if len(clusters) == 0 {
		t.Fatalf("expected starred event clusters, got 0")
	}
}

func TestLoadEventGroupsForCategory_Integration(t *testing.T) {
	repo := setupSeededDB(t)

	groups, err := repo.LoadEventGroupsForCategory("BGM", 2026)
	if err != nil {
		t.Fatalf("LoadEventGroupsForCategory failed: %v", err)
	}

	if len(groups) == 0 {
		t.Fatalf("expected event groups for BGM, got 0")
	}
}

func TestSearchEvents_OnlyFree_Integration(t *testing.T) {
	repo := setupSeededDB(t)

	// Search for RPG events for leader@example.com
	// RPG has:
	// - RPG26ND200001 (D&D Epic, Jul 30 13:00-17:00). Overlaps with leader's purchased BGM26ND100002 (Jul 30 14:00-16:00).
	// - RPG26ND200002 (D&D Intro, Jul 31 13:00-17:00). No overlap.
	
	// 1. Without OnlyFree filter: should return both RPG events
	qNoFree := postgres.SearchQuery{
		CategoryShortCode: "RPG",
		Year:              2026,
		OnlyFree:          false,
	}
	groupsNoFree, err := repo.SearchEvents(qNoFree)
	if err != nil {
		t.Fatalf("SearchEvents failed: %v", err)
	}
	
	if len(groupsNoFree) != 2 {
		t.Errorf("expected 2 RPG event groups, got %d", len(groupsNoFree))
	}

	// 2. With OnlyFree filter for leader@example.com: should filter out RPG26ND200001 and only return RPG26ND200002
	qFree := postgres.SearchQuery{
		CategoryShortCode: "RPG",
		Year:              2026,
		OnlyFree:          true,
		UserEmail:         "leader@example.com",
	}
	groupsFree, err := repo.SearchEvents(qFree)
	if err != nil {
		t.Fatalf("SearchEvents failed: %v", err)
	}

	// The Catan ticket overlaps with D&D Epic (RPG26ND200001), so only RPG26ND200002 should remain.
	if len(groupsFree) != 1 {
		t.Errorf("expected 1 RPG event group (no overlap), got %d", len(groupsFree))
	} else if groupsFree[0].Name != "Dungeons & Dragons Epic Intro" {
		t.Errorf("expected Dungeons & Dragons Epic Intro (no overlap), got %s", groupsFree[0].Name)
	}
}
