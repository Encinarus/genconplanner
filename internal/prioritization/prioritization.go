package prioritization

import (
	"fmt"
	"sort"

	"github.com/Encinarus/genconplanner/internal/events"
	"github.com/Encinarus/genconplanner/internal/postgres"
)

type WishlistItem struct {
	Event     *events.GenconEvent
	Status    string // "Primary" or "Backup"
	Reasoning []string
	Score     float64
}

type groupStats struct {
	TotalTickets int
	NumSessions  int
	MaxTier      string
	ClusterKey   string
}

func getClusterKey(e *events.GenconEvent) string {
	return fmt.Sprintf("%s|%s|%s", e.Title, e.ShortDescription, e.ShortCategory)
}

func tierToScore(tier string) float64 {
	switch tier {
	case "must_have":
		return 10000
	case "very_interested":
		return 1000
	case "somewhat_interested":
		return 100
	default:
		return 0
	}
}

func GeneratePersonalWishlist(starred []postgres.StarredEvent, allEvents []*events.GenconEvent) []WishlistItem {
	// 1. Map events by ID for quick lookup
	eventMap := make(map[string]*events.GenconEvent)
	for _, e := range allEvents {
		eventMap[e.EventId] = e
	}

	// 2. Map starred events and calculate group stats
	starredMap := make(map[string]postgres.StarredEvent)
	groupStatsMap := make(map[string]*groupStats)

	for _, se := range starred {
		starredMap[se.EventId] = se
		event, found := eventMap[se.EventId]
		if !found {
			continue
		}

		key := getClusterKey(event)
		stats, found := groupStatsMap[key]
		if !found {
			stats = &groupStats{ClusterKey: key}
			groupStatsMap[key] = stats
		}
		stats.NumSessions++
		stats.TotalTickets += event.TicketsAvailable
		if tierToScore(se.Tier) > tierToScore(stats.MaxTier) {
			stats.MaxTier = se.Tier
		}
	}

	// 3. Score each session
	type scoredSession struct {
		Event      *events.GenconEvent
		Score      float64
		Reasoning  []string
		ClusterKey string
	}
	var scoredSessions []scoredSession

	for _, se := range starred {
		event, found := eventMap[se.EventId]
		if !found {
			continue
		}
		key := getClusterKey(event)
		stats := groupStatsMap[key]

		score := tierToScore(se.Tier)
		reasoning := []string{}

		if se.Tier == "must_have" {
			reasoning = append(reasoning, "Must Have")
		} else if se.Tier == "very_interested" {
			reasoning = append(reasoning, "Very Interested")
		}

		// Scarcity Bonus (Group level)
		scarcityBonus := 0.0
		if stats.TotalTickets > 0 {
			scarcityBonus = 5000.0 / float64(stats.TotalTickets+1)
		} else {
			scarcityBonus = 5000.0 // Extremely rare/sold out?
		}
		
		if scarcityBonus > 1000 {
			reasoning = append(reasoning, "Rare Event")
		}
		score += scarcityBonus

		// Session availability bonus
		if stats.NumSessions == 1 {
			score += 1000
			reasoning = append(reasoning, "Single Session")
		}

		scoredSessions = append(scoredSessions, scoredSession{
			Event:      event,
			Score:      score,
			Reasoning:  reasoning,
			ClusterKey: key,
		})
	}

	// Sort groups by their best session's score
	type groupPriority struct {
		ClusterKey string
		MaxScore   float64
	}
	var groupPriorities []groupPriority
	for key := range groupStatsMap {
		maxScore := 0.0
		for _, s := range scoredSessions {
			if s.ClusterKey == key && s.Score > maxScore {
				maxScore = s.Score
			}
		}
		groupPriorities = append(groupPriorities, groupPriority{key, maxScore})
	}
	sort.Slice(groupPriorities, func(i, j int) bool {
		return groupPriorities[i].MaxScore > groupPriorities[j].MaxScore
	})

	// 4. Pass 1: Perfect Calendar (Primary)
	// Greedy selection: highest priority group gets its best fitting session.
	var wishlist []WishlistItem
	selectedGroups := make(map[string]int) // ClusterKey -> Count
	
	// Helper to check for conflicts
	hasConflict := func(e1 *events.GenconEvent, currentList []WishlistItem) bool {
		for _, item := range currentList {
			if item.Status != "Primary" {
				continue
			}
			e2 := item.Event
			// Standard interval overlap check
			if e1.StartTime.Before(e2.EndTime) && e2.StartTime.Before(e1.EndTime) {
				return true
			}
		}
		return false
	}

	// Pass 1: One session per group, prioritizing higher groups
	for _, gp := range groupPriorities {
		// Find sessions for this group that don't conflict with current wishlist
		type candidateSession struct {
			session       *scoredSession
			conflictCount int
		}
		var candidates []candidateSession

		for i := range scoredSessions {
			s := &scoredSessions[i]
			if s.ClusterKey != gp.ClusterKey {
				continue
			}
			if !hasConflict(s.Event, wishlist) {
				// Count how many OTHER starred sessions this would conflict with
				conflicts := 0
				for j := range scoredSessions {
					other := &scoredSessions[j]
					if other.ClusterKey == gp.ClusterKey {
						continue
					}
					// Interval overlap check
					if s.Event.StartTime.Before(other.Event.EndTime) && other.Event.StartTime.Before(s.Event.EndTime) {
						conflicts++
					}
				}
				candidates = append(candidates, candidateSession{s, conflicts})
			}
		}

		if len(candidates) > 0 {
			// Sort candidates:
			// 1. Higher Session Score
			// 2. Lower Conflict Count
			sort.Slice(candidates, func(i, j int) bool {
				if candidates[i].session.Score != candidates[j].session.Score {
					return candidates[i].session.Score > candidates[j].session.Score
				}
				return candidates[i].conflictCount < candidates[j].conflictCount
			})

			best := candidates[0]
			wishlist = append(wishlist, WishlistItem{
				Event:     best.session.Event,
				Status:    "Primary",
				Reasoning: append(best.session.Reasoning, "Perfect Fit"),
				Score:     best.session.Score,
			})
			selectedGroups[gp.ClusterKey]++
		}
	}

	// 5. Pass 2: Backups
	// Fill remaining up to 50 items or 3 per group
	// Sort all remaining sessions by score
	sort.Slice(scoredSessions, func(i, j int) bool {
		return scoredSessions[i].Score > scoredSessions[j].Score
	})

	for _, s := range scoredSessions {
		if len(wishlist) >= 50 {
			break
		}
		
		// Check if we already have this specific event in wishlist
		alreadyIn := false
		for _, item := range wishlist {
			if item.Event.EventId == s.Event.EventId {
				alreadyIn = true
				break
			}
		}
		if alreadyIn {
			continue
		}

		// Cap at 3 per group
		if selectedGroups[s.ClusterKey] >= 3 {
			continue
		}

		wishlist = append(wishlist, WishlistItem{
			Event:     s.Event,
			Status:    "Backup",
			Reasoning: append(s.Reasoning, "Backup Option"),
			Score:     s.Score,
		})
		selectedGroups[s.ClusterKey]++
	}

	return wishlist
}
