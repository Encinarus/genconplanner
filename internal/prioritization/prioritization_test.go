package prioritization

import (
	"testing"
	"time"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
)

func TestGeneratePersonalWishlist(t *testing.T) {
	now := time.Now()
	
	// Group A: Must have, two sessions
	eA1 := &events.GenconEvent{
		EventId: "A1", Title: "Group A", StartTime: now, EndTime: now.Add(time.Hour), TicketsAvailable: 10,
	}
	eA2 := &events.GenconEvent{
		EventId: "A2", Title: "Group A", StartTime: now.Add(2 * time.Hour), EndTime: now.Add(3 * time.Hour), TicketsAvailable: 10,
	}
	
	// Group B: Very interested, conflicts with A1
	eB1 := &events.GenconEvent{
		EventId: "B1", Title: "Group B", StartTime: now, EndTime: now.Add(time.Hour), TicketsAvailable: 5,
	}
	
	// Group C: Must have, single session, rare
	eC1 := &events.GenconEvent{
		EventId: "C1", Title: "Group C", StartTime: now.Add(4 * time.Hour), EndTime: now.Add(5 * time.Hour), TicketsAvailable: 2,
	}

	allEvents := []*events.GenconEvent{eA1, eA2, eB1, eC1}
	starred := []postgres.StarredEvent{
		{EventId: "A1", Tier: "must_have"},
		{EventId: "A2", Tier: "must_have"},
		{EventId: "B1", Tier: "very_interested"},
		{EventId: "C1", Tier: "must_have"},
	}

	wishlist := GeneratePersonalWishlist(starred, allEvents)

	if len(wishlist) == 0 {
		t.Fatal("Expected non-empty wishlist")
	}

	// Verify priority
	// Group C should be high because it's Must Have + Rare + Single Session
	if wishlist[0].Event.EventId != "C1" {
		t.Errorf("Expected C1 to be #1, got %s", wishlist[0].Event.EventId)
	}

	// Group A should have one session in Primary.
	// Between A1 and A2, they are equal tier. 
	// A1 conflicts with B1? No, B1 is lower priority.
	
	primaryCount := 0
	for _, item := range wishlist {
		if item.Status == "Primary" {
			primaryCount++
		}
	}
	
	if primaryCount < 3 {
		t.Errorf("Expected at least 3 primary items, got %d", primaryCount)
	}
}

func TestAntiSpam(t *testing.T) {
	now := time.Now()
	allEvents := []*events.GenconEvent{}
	starred := []postgres.StarredEvent{}
	
	for i := 0; i < 10; i++ {
		id := string(rune('0' + i))
		e := &events.GenconEvent{
			EventId: id, Title: "Spam Group", StartTime: now.Add(time.Duration(i) * time.Hour), EndTime: now.Add(time.Duration(i+1) * time.Hour), TicketsAvailable: 10,
		}
		allEvents = append(allEvents, e)
		starred = append(starred, postgres.StarredEvent{EventId: id, Tier: "must_have"})
	}

	wishlist := GeneratePersonalWishlist(starred, allEvents)
	
	groupCount := 0
	for _, item := range wishlist {
		if item.Event.Title == "Spam Group" {
			groupCount++
		}
	}
	
	if groupCount > 3 {
		t.Errorf("Expected at most 3 items from Spam Group, got %d", groupCount)
	}
}
