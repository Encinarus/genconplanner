package prioritization

import (
	"fmt"
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

	wishlist := GeneratePersonalWishlist(starred, allEvents, []postgres.WishlistConstraint{})

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

	wishlist := GeneratePersonalWishlist(starred, allEvents, []postgres.WishlistConstraint{})
	
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

func TestExclusiveBlockedTimes(t *testing.T) {
	// Mock constraints
	constraints := []postgres.WishlistConstraint{
		{
			DayOfWeek:   int(time.Thursday),
			StartHour:   8,
			StartMinute: 0,
			EndHour:     8,
			EndMinute:   30,
		},
		{
			DayOfWeek:   -1, // Every Day
			StartHour:   23,
			StartMinute: 0,
			EndHour:     1,
			EndMinute:   0,
		},
	}

	testCases := []struct {
		name     string
		start    time.Time
		duration int
		blocked  bool
	}{
		{
			name:     "Event ending at block start (8:00)",
			start:    time.Date(2026, 7, 30, 6, 0, 0, 0, events.INDIANAPOLIS), // Thursday
			duration: 120, // 6:00 to 8:00
			blocked:  false,
		},
		{
			name:     "Event starting at block end (8:30)",
			start:    time.Date(2026, 7, 30, 8, 30, 0, 0, events.INDIANAPOLIS),
			duration: 30, // 8:30 to 9:00
			blocked:  false,
		},
		{
			name:     "Event overlapping middle (8:15)",
			start:    time.Date(2026, 7, 30, 8, 15, 0, 0, events.INDIANAPOLIS),
			duration: 30, // 8:15 to 8:45
			blocked:  true,
		},
		{
			name:     "Event exactly matching block (8:00-8:30)",
			start:    time.Date(2026, 7, 30, 8, 0, 0, 0, events.INDIANAPOLIS),
			duration: 30,
			blocked:  true,
		},
		{
			name:     "Event spanning across block (7:45-8:45)",
			start:    time.Date(2026, 7, 30, 7, 45, 0, 0, events.INDIANAPOLIS),
			duration: 60,
			blocked:  true,
		},
		{
			name:     "Midnight block boundary start (23:00)",
			start:    time.Date(2026, 7, 30, 22, 0, 0, 0, events.INDIANAPOLIS),
			duration: 60, // 22:00 to 23:00
			blocked:  false,
		},
		{
			name:     "Midnight block interior (00:00)",
			start:    time.Date(2026, 7, 30, 23, 30, 0, 0, events.INDIANAPOLIS),
			duration: 60, // 23:30 to 00:30
			blocked:  true,
		},
		{
			name:     "Midnight block boundary end (01:00)",
			start:    time.Date(2026, 7, 31, 1, 0, 0, 0, events.INDIANAPOLIS),
			duration: 60, // 01:00 to 02:00
			blocked:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &events.GenconEvent{
				StartTime: tc.start,
				Duration:  tc.duration,
				EndTime:   tc.start.Add(time.Duration(tc.duration) * time.Minute),
			}

			isBlocked := OverlapsBlockedTime(event, constraints)
			if isBlocked != tc.blocked {
				t.Errorf("Expected blocked: %v, Got: %v", tc.blocked, isBlocked)
			}
		})
	}
}

func TestFlexibleBlockedTimes(t *testing.T) {
	testCases := []struct {
		name        string
		constraints []postgres.WishlistConstraint
		candidate   *events.GenconEvent
		primary     []WishlistItem
		shouldPass  bool
	}{
		{
			name: "Single event leaving large gap (10-12, gap 12-14)",
			constraints: []postgres.WishlistConstraint{
				{
					DayOfWeek:          int(time.Thursday),
					StartHour:          10,
					StartMinute:        0,
					EndHour:            14,
					EndMinute:          0,
					MinDurationMinutes: 60,
				},
			},
			candidate: &events.GenconEvent{
				StartTime: time.Date(2026, 7, 30, 10, 0, 0, 0, events.INDIANAPOLIS),
				Duration:  120, // 10:00 to 12:00
				EndTime:   time.Date(2026, 7, 30, 12, 0, 0, 0, events.INDIANAPOLIS),
			},
			primary:    []WishlistItem{},
			shouldPass: true,
		},
		{
			name: "Event too long for window (10-13:30, gap only 30m)",
			constraints: []postgres.WishlistConstraint{
				{
					DayOfWeek:          int(time.Thursday),
					StartHour:          10,
					StartMinute:        0,
					EndHour:            14,
					EndMinute:          0,
					MinDurationMinutes: 60,
				},
			},
			candidate: &events.GenconEvent{
				StartTime: time.Date(2026, 7, 30, 10, 0, 0, 0, events.INDIANAPOLIS),
				Duration:  210, // 10:00 to 13:30
				EndTime:   time.Date(2026, 7, 30, 13, 30, 0, 0, events.INDIANAPOLIS),
			},
			primary:    []WishlistItem{},
			shouldPass: false,
		},
		{
			name: "Two events leaving gap in middle (10-11 and 12-14, gap 11-12)",
			constraints: []postgres.WishlistConstraint{
				{
					DayOfWeek:          int(time.Thursday),
					StartHour:          10,
					StartMinute:        0,
					EndHour:            14,
					EndMinute:          0,
					MinDurationMinutes: 60,
				},
			},
			candidate: &events.GenconEvent{
				StartTime: time.Date(2026, 7, 30, 10, 0, 0, 0, events.INDIANAPOLIS),
				Duration:  60, // 10:00 to 11:00
				EndTime:   time.Date(2026, 7, 30, 11, 0, 0, 0, events.INDIANAPOLIS),
			},
			primary: []WishlistItem{
				{
					Status: "Primary",
					Event: &events.GenconEvent{
						StartTime: time.Date(2026, 7, 30, 12, 0, 0, 0, events.INDIANAPOLIS),
						Duration:  120, // 12:00 to 14:00
						EndTime:   time.Date(2026, 7, 30, 14, 0, 0, 0, events.INDIANAPOLIS),
					},
				},
			},
			shouldPass: true,
		},
		{
			name: "Three events breaking all gaps (10:30-11, 11:30-12, 12:30-13:30)",
			constraints: []postgres.WishlistConstraint{
				{
					DayOfWeek:          int(time.Thursday),
					StartHour:          10,
					StartMinute:        0,
					EndHour:            14,
					EndMinute:          0,
					MinDurationMinutes: 60,
				},
			},
			candidate: &events.GenconEvent{
				StartTime: time.Date(2026, 7, 30, 11, 30, 0, 0, events.INDIANAPOLIS),
				Duration:  30, // 11:30 to 12:00
				EndTime:   time.Date(2026, 7, 30, 12, 0, 0, 0, events.INDIANAPOLIS),
			},
			primary: []WishlistItem{
				{
					Status: "Primary",
					Event: &events.GenconEvent{
						StartTime: time.Date(2026, 7, 30, 10, 30, 0, 0, events.INDIANAPOLIS),
						Duration:  30, // 10:30 to 11:00
						EndTime:   time.Date(2026, 7, 30, 11, 0, 0, 0, events.INDIANAPOLIS),
					},
				},
				{
					Status: "Primary",
					Event: &events.GenconEvent{
						StartTime: time.Date(2026, 7, 30, 12, 30, 0, 0, events.INDIANAPOLIS),
						Duration:  60, // 12:30 to 13:30
						EndTime:   time.Date(2026, 7, 30, 13, 30, 0, 0, events.INDIANAPOLIS),
					},
				},
			},
			shouldPass: false,
		},
		{
			name: "Permissive start/end bounds (7-8 and 8:30-9, gap 8-8:30)",
			constraints: []postgres.WishlistConstraint{
				{
					DayOfWeek:          int(time.Thursday),
					StartHour:          7,
					StartMinute:        0,
					EndHour:            9,
					EndMinute:          0,
					MinDurationMinutes: 30,
				},
			},
			candidate: &events.GenconEvent{
				StartTime: time.Date(2026, 7, 30, 8, 30, 0, 0, events.INDIANAPOLIS),
				Duration:  30, // 8:30 to 9:00
				EndTime:   time.Date(2026, 7, 30, 9, 0, 0, 0, events.INDIANAPOLIS),
			},
			primary: []WishlistItem{
				{
					Status: "Primary",
					Event: &events.GenconEvent{
						StartTime: time.Date(2026, 7, 30, 7, 0, 0, 0, events.INDIANAPOLIS),
						Duration:  60, // 7:00 to 8:00
						EndTime:   time.Date(2026, 7, 30, 8, 0, 0, 0, events.INDIANAPOLIS),
					},
				},
			},
			shouldPass: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate wishlist with these constraints
			// To simplify, we just call the underlying logic if we can, 
			// but since it's an anonymous function in GeneratePersonalWishlist, 
			// we have to call the whole thing.
			
			starred := []postgres.StarredEvent{
				{EventId: "CANDIDATE", Tier: "must_have"},
			}
			tc.candidate.EventId = "CANDIDATE"
			tc.candidate.Title = "Candidate"
			tc.candidate.ShortCategory = "RPG"
			allEvents := []*events.GenconEvent{tc.candidate}

			for i, p := range tc.primary {
				eventId := fmt.Sprintf("PRIMARY_%d", i)
				starred = append(starred, postgres.StarredEvent{
					EventId: eventId,
					Tier:    "must_have",
				})
				p.Event.EventId = eventId
				p.Event.Title = fmt.Sprintf("A_Primary %d", i) // Prioritize over Candidate
				p.Event.ShortCategory = "RPG"
				allEvents = append(allEvents, p.Event)
			}

			wishlist := GeneratePersonalWishlist(starred, allEvents, tc.constraints)
			
			foundCandidate := false
			for _, item := range wishlist {
				if item.Event.EventId == "CANDIDATE" && item.Status == "Primary" {
					foundCandidate = true
					break
				}
			}

			if foundCandidate != tc.shouldPass {
				t.Errorf("Expected candidate Primary status to be %v, got %v", tc.shouldPass, foundCandidate)
			}
		})
	}
}

func TestRearrangementPass(t *testing.T) {
	now := time.Date(2026, 7, 30, 0, 0, 0, 0, events.INDIANAPOLIS) // Thursday midnight

	// Group A: Must have. Two sessions available.
	// A1: 9:30pm (21:30) to 11:30pm (23:30) - Late night
	eA1 := &events.GenconEvent{
		EventId: "A1", Title: "Group A", StartTime: now.Add(21*time.Hour + 30*time.Minute), EndTime: now.Add(23*time.Hour + 30*time.Minute), TicketsAvailable: 10,
	}
	// A2: 12:00pm (12:00) to 2:00pm (14:00) - Peak hours (10am-5pm)
	eA2 := &events.GenconEvent{
		EventId: "A2", Title: "Group A", StartTime: now.Add(12 * time.Hour), EndTime: now.Add(14 * time.Hour), TicketsAvailable: 10,
	}

	allEvents := []*events.GenconEvent{eA1, eA2}
	starred := []postgres.StarredEvent{
		{EventId: "A1", Tier: "must_have"},
		{EventId: "A2", Tier: "must_have"},
	}

	wishlist := GeneratePersonalWishlist(starred, allEvents, []postgres.WishlistConstraint{})

	if len(wishlist) == 0 {
		t.Fatal("Expected non-empty wishlist")
	}

	// Because A1 < A2, Pass 1 initially picks A1 due to tie-breaking.
	// But the Rearrangement Pass should swap it for A2 because A2 falls in peak hours!
	if wishlist[0].Event.EventId != "A2" {
		t.Errorf("Expected A2 (peak hours) to be selected, got %s", wishlist[0].Event.EventId)
	}
}
