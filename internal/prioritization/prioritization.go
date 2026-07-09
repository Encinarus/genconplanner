package prioritization

import (
	"fmt"
	"sort"

	"time"

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
	case "purchased":
		return 50000
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

func IsTimeBlocked(checkTime time.Time, constraints []postgres.WishlistConstraint) bool {
	dow := int(checkTime.Weekday())
	totalMinutes := checkTime.Hour()*60 + checkTime.Minute()

	for _, c := range constraints {
		// Check day
		if c.DayOfWeek != -1 && c.DayOfWeek != dow {
			continue
		}

		if c.MinDurationMinutes > 0 {
			continue
		}

		startTotal := c.StartHour*60 + c.StartMinute
		endTotal := c.EndHour*60 + c.EndMinute

		if startTotal == endTotal {
			continue
		}

		if startTotal < endTotal {
			if totalMinutes > startTotal && totalMinutes < endTotal {
				return true
			}
		} else {
			// Crosses midnight within its own context (usually for "Every Day")
			if totalMinutes > startTotal || totalMinutes < endTotal {
				return true
			}
		}
	}
	return false
}

func OverlapsBlockedTime(e *events.GenconEvent, constraints []postgres.WishlistConstraint) bool {
	if len(constraints) == 0 {
		return false
	}
	// Check each 15-min interval of the event for precision
	for m := 0; m <= e.Duration; m += 15 {
		checkTime := e.StartTime.Add(time.Duration(m) * time.Minute)
		if IsTimeBlocked(checkTime, constraints) {
			return true
		}
	}
	// Also check final end time
	if IsTimeBlocked(e.EndTime, constraints) {
		return true
	}
	return false
}

func overlapsAnyPurchased(e *events.GenconEvent, purchased []*events.GenconEvent) bool {
	for _, p := range purchased {
		if e.EventId == p.EventId {
			continue
		}
		if e.StartTime.Before(p.EndTime) && p.StartTime.Before(e.EndTime) {
			return true
		}
	}
	return false
}

func GeneratePersonalWishlist(starred []postgres.StarredEvent, allEvents []*events.GenconEvent, constraints []postgres.WishlistConstraint, partyPurchases map[string]int) []WishlistItem {
	// 1. Map events by ID for quick lookup
	eventMap := make(map[string]*events.GenconEvent)
	for _, e := range allEvents {
		eventMap[e.EventId] = e
	}

	// 1.5 Identify purchased events and their cluster keys
	var purchasedEvents []*events.GenconEvent
	purchasedGroups := make(map[string]bool)
	for _, se := range starred {
		if se.Tier == "purchased" {
			if event, found := eventMap[se.EventId]; found {
				purchasedEvents = append(purchasedEvents, event)
				purchasedGroups[getClusterKey(event)] = true
			}
		}
	}

	// 2. Map starred events and calculate group stats
	starredMap := make(map[string]postgres.StarredEvent)
	groupStatsMap := make(map[string]*groupStats)

	for _, se := range starred {
		if se.Tier == "not_interested" {
			continue
		}
		starredMap[se.EventId] = se
		event, found := eventMap[se.EventId]
		if !found {
			continue
		}
		if se.Tier != "purchased" && event.TicketsAvailable <= 0 {
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
		if se.Tier == "not_interested" {
			continue
		}
		event, found := eventMap[se.EventId]
		if !found {
			continue
		}
		if se.Tier != "purchased" && event.TicketsAvailable <= 0 {
			continue
		}
		key := getClusterKey(event)
		stats := groupStatsMap[key]

		score := tierToScore(se.Tier)
		reasoning := []string{}

		if se.Tier == "purchased" {
			reasoning = append(reasoning, "Purchased")
		} else if se.Tier == "must_have" {
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

		// Tickets remaining boost (more tickets being better)
		if event.TicketsAvailable > 0 {
			score += float64(event.TicketsAvailable) * 10.0
			reasoning = append(reasoning, fmt.Sprintf("%d Tickets Left", event.TicketsAvailable))
		}

		// Party member purchases boost
		if partyPurchases != nil {
			count := partyPurchases[event.EventId]
			if count > 0 {
				score += float64(count) * 2000.0
				if count == 1 {
					reasoning = append(reasoning, "1 Party Member Purchased")
				} else {
					reasoning = append(reasoning, fmt.Sprintf("%d Party Members Purchased", count))
				}
			}
		}

		// Apply Time Constraints
		if len(constraints) > 0 && OverlapsBlockedTime(event, constraints) {
			score -= 1000000 // Massive penalty for blocked times
			reasoning = append(reasoning, "Blocked Time")
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
		if groupPriorities[i].MaxScore != groupPriorities[j].MaxScore {
			return groupPriorities[i].MaxScore > groupPriorities[j].MaxScore
		}
		return groupPriorities[i].ClusterKey < groupPriorities[j].ClusterKey
	})

	// 4. Pass 1: Perfect Calendar (Primary)
	// Greedy selection: highest priority group gets its best fitting session.
	var wishlist []WishlistItem
	selectedGroups := make(map[string]int) // ClusterKey -> Count

	// Helper to check for gaps in flexible constraints
	checkFlexibleConstraints := func(candidate *events.GenconEvent, currentList []WishlistItem) bool {
		dow := int(candidate.StartTime.Weekday())
		for _, c := range constraints {
			if c.MinDurationMinutes <= 0 {
				continue
			}
			if c.DayOfWeek != -1 && c.DayOfWeek != dow {
				continue
			}

			windowStart := c.StartHour*60 + c.StartMinute
			windowEnd := c.EndHour*60 + c.EndMinute
			if windowEnd <= windowStart {
				// For now, only support windows within a single day for flexible breaks.
				// Supporting cross-midnight flexible breaks is significantly more complex.
				continue
			}

			type interval struct{ start, end int }
			var occupied []interval

			addEvent := func(e *events.GenconEvent) {
				if int(e.StartTime.Weekday()) != dow {
					return
				}
				eStart := e.StartTime.Hour()*60 + e.StartTime.Minute()
				eEnd := e.EndTime.Hour()*60 + e.EndTime.Minute()

				// Overlap check and clipping
				if eStart < windowEnd && eEnd > windowStart {
					clipStart := eStart
					if clipStart < windowStart {
						clipStart = windowStart
					}
					clipEnd := eEnd
					if clipEnd > windowEnd {
						clipEnd = windowEnd
					}
					if clipStart < clipEnd {
						occupied = append(occupied, interval{clipStart, clipEnd})
					}
				}
			}

			addEvent(candidate)
			for _, item := range currentList {
				if item.Status == "Primary" {
					addEvent(item.Event)
				}
			}

			sort.Slice(occupied, func(i, j int) bool {
				return occupied[i].start < occupied[j].start
			})

			// Check for gap
			lastEnd := windowStart
			foundGap := false
			for _, inter := range occupied {
				if inter.start-lastEnd >= c.MinDurationMinutes {
					foundGap = true
					break
				}
				if inter.end > lastEnd {
					lastEnd = inter.end
				}
			}
			if !foundGap && windowEnd-lastEnd >= c.MinDurationMinutes {
				foundGap = true
			}

			if !foundGap {
				return false
			}
		}
		return true
	}

	// Helper to check for conflicts
	hasConflict := func(e1 *events.GenconEvent, currentList []WishlistItem) bool {
		if overlapsAnyPurchased(e1, purchasedEvents) {
			return true
		}
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

		return !checkFlexibleConstraints(e1, currentList)
	}

	// Pass 1: One session per group, prioritizing higher groups
	for _, gp := range groupPriorities {
		if purchasedGroups[gp.ClusterKey] {
			// Mark this group as fully satisfied so no sessions of this game are ever added to the wishlist
			selectedGroups[gp.ClusterKey] = 999
			continue
		}
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
			if !hasConflict(s.Event, wishlist) && s.Score > 0 {
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
			// 3. Deterministic tie-breaker (EventId)
			sort.Slice(candidates, func(i, j int) bool {
				if candidates[i].session.Score != candidates[j].session.Score {
					return candidates[i].session.Score > candidates[j].session.Score
				}
				if candidates[i].conflictCount != candidates[j].conflictCount {
					return candidates[i].conflictCount < candidates[j].conflictCount
				}
				return candidates[i].session.Event.EventId < candidates[j].session.Event.EventId
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

	// 4.5 Rearrangement Pass: Optimize Perfect Calendar for Peak Hours (10am-5pm) and Travel Gaps (>= 10 mins)
	if len(wishlist) > 0 {
		calcCalendarScore := func(calendar []WishlistItem) float64 {
			score := 0.0
			// Base score
			for _, item := range calendar {
				score += item.Score
			}

			// Peak Hours Bonus (10am to 5pm is 600 to 1020 minutes from midnight)
			for _, item := range calendar {
				e := item.Event
				startMins := e.StartTime.Hour()*60 + e.StartTime.Minute()
				endMins := e.EndTime.Hour()*60 + e.EndTime.Minute()

				peakStart := startMins
				if peakStart < 600 {
					peakStart = 600
				}
				peakEnd := endMins
				if peakEnd > 1020 {
					peakEnd = 1020
				}
				if peakStart < peakEnd {
					score += float64(peakEnd - peakStart) // +1 point per minute in peak hours
				}

				// Off-Peak Penalty (before 9am [540 mins] or after 7pm [1140 mins])
				if startMins < 540 {
					before9End := endMins
					if before9End > 540 {
						before9End = 540
					}
					score -= float64(before9End-startMins) * 2.0 // -2 points per minute before 9am
				}

				if endMins > 1140 {
					after7Start := startMins
					if after7Start < 1140 {
						after7Start = 1140
					}
					score -= float64(endMins-after7Start) * 2.0 // -2 points per minute after 7pm
				}
			}

			// Gap Bonus (>= 10 mins gap between adjacent events on the same day)
			eventsByDay := make(map[int][]*events.GenconEvent)
			for _, item := range calendar {
				dow := int(item.Event.StartTime.Weekday())
				eventsByDay[dow] = append(eventsByDay[dow], item.Event)
			}

			for _, dayEvents := range eventsByDay {
				if len(dayEvents) < 2 {
					continue
				}
				sort.Slice(dayEvents, func(i, j int) bool {
					return dayEvents[i].StartTime.Before(dayEvents[j].StartTime)
				})

				for i := 0; i < len(dayEvents)-1; i++ {
					gap := dayEvents[i+1].StartTime.Sub(dayEvents[i].EndTime).Minutes()
					if gap >= 10 {
						score += 50.0 // +50 points for a healthy travel gap
					}
				}
			}

			return score
		}

		currentScore := calcCalendarScore(wishlist)
		improved := true
		iterations := 0

		for improved && iterations < 10 {
			improved = false
			iterations++

			for i := range wishlist {
				targetItem := wishlist[i]
				targetKey := getClusterKey(targetItem.Event)

				if purchasedGroups[targetKey] {
					continue // do not rearrange purchased events!
				}

				// Create temp calendar without targetItem
				var tempCalendar []WishlistItem
				for j := range wishlist {
					if j != i {
						tempCalendar = append(tempCalendar, wishlist[j])
					}
				}

				bestCandidate := targetItem
				bestScore := currentScore

				// Find all candidate sessions for this group
				for k := range scoredSessions {
					s := &scoredSessions[k]
					if s.ClusterKey != targetKey || s.Score <= 0 {
						continue
					}

					if !hasConflict(s.Event, tempCalendar) {
						candidateItem := WishlistItem{
							Event:     s.Event,
							Status:    "Primary",
							Reasoning: append(s.Reasoning, "Perfect Fit"),
							Score:     s.Score,
						}
						testCalendar := append(append([]WishlistItem(nil), tempCalendar...), candidateItem)
						testScore := calcCalendarScore(testCalendar)

						if testScore > bestScore {
							bestScore = testScore
							bestCandidate = candidateItem
							improved = true
						}
					}
				}

				if improved && bestCandidate.Event.EventId != targetItem.Event.EventId {
					wishlist[i] = bestCandidate
					currentScore = bestScore
				}
			}
		}
	}

	// 5. Pass 2: Backups
	// Fill remaining up to 50 items or 3 per group
	// We'll pick them one by one to dynamically account for overlaps
	for len(wishlist) < 50 {
		var bestSession *scoredSession
		var bestScoreWithPenalty float64 = -2000000.0 // Start very low

		for i := range scoredSessions {
			s := &scoredSessions[i]

			if overlapsAnyPurchased(s.Event, purchasedEvents) {
				continue
			}

			// 1. Check if specific event ID is already in wishlist
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

			// 2. Cap at 3 sessions per group (including primary)
			if selectedGroups[s.ClusterKey] >= 3 {
				continue
			}

			// 3. Calculate overlap penalty to "spread out" backups
			overlapPenalty := 0.0
			for _, item := range wishlist {
				// Interval overlap check
				if s.Event.StartTime.Before(item.Event.EndTime) && item.Event.StartTime.Before(s.Event.EndTime) {
					if item.Status == "Primary" {
						overlapPenalty += 2000 // Higher penalty for overlapping the primary schedule
					} else {
						overlapPenalty += 1000 // Lower penalty for overlapping another backup
					}

					// Extra penalty if it overlaps with a session of the SAME game
					if getClusterKey(item.Event) == s.ClusterKey {
						overlapPenalty += 5000
					}
				}
			}

			// Flexible block penalty
			if !checkFlexibleConstraints(s.Event, wishlist) {
				overlapPenalty += 5000 // High penalty for breaking a flexible break
			}

			scoreWithPenalty := s.Score - overlapPenalty
			if scoreWithPenalty > bestScoreWithPenalty {
				bestScoreWithPenalty = scoreWithPenalty
				bestSession = s
			}
		}

		if bestSession == nil || bestScoreWithPenalty < -500000 {
			// No more viable sessions (the -500k check avoids picking "Blocked Time" sessions)
			break
		}

		wishlist = append(wishlist, WishlistItem{
			Event:     bestSession.Event,
			Status:    "Backup",
			Reasoning: append(bestSession.Reasoning, "Backup Option"),
			Score:     bestSession.Score,
		})
		selectedGroups[bestSession.ClusterKey]++
	}

	return wishlist
}
