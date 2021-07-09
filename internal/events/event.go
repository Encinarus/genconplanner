package events

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func PartitionEventsByDay(loadedEvents []*GenconEvent) map[string][]*GenconEvent {
	eventsPerDay := make(map[string][]*GenconEvent)

	for _, event := range loadedEvents {
		day := event.StartTime.Weekday().String()
		eventsPerDay[day] = append(eventsPerDay[day], event)
	}

	return eventsPerDay
}

func PartitionEventsByCategory(loadedEvents []*GenconEvent) map[string][]*GenconEvent {
	eventsPerCategory := make(map[string][]*GenconEvent)

	for _, event := range loadedEvents {
		category := event.ShortCategory
		eventsPerCategory[category] = append(eventsPerCategory[category], event)
	}

	return eventsPerCategory
}

func AllCategories() map[string]string {
	return map[string]string{
		"ANI":  "Anime Activities",
		"BGM":  "Board Games",
		"CGM":  "Non-Collectable/Tradable Card Games",
		"EGM":  "Electronic Games",
		"ENT":  "Entertainment Events",
		"FLM":  "Film Fest",
		"HMN":  "Historical Miniatures",
		"KID":  "Kids Activities",
		"LRP":  "Larps",
		"MHE":  "Miniature Hobby Events",
		"NMN":  "Non-Historical Miniatures",
		"RPG":  "Role Playing Games",
		"RPGA": "Role Playing Game Association",
		"SEM":  "Seminiars",
		"SPA":  "Spousal Activities",
		"TCG":  "Tradeable Card Game",
		"TDA":  "True Dungeon",
		"TRD":  "Trade Day Events",
		"WKS":  "Workshop",
		"ZED":  "Isle of Misfit Events",
	}
}

func LongCategory(shortCategory string) string {
	longCat, found := AllCategories()[shortCategory]

	if found {
		return longCat
	} else {
		return shortCategory
	}
}

func CategoryFromEvent(rawEventId string) string {
	category, _ := splitId(rawEventId)
	return category
}

func YearFromEvent(rawEventId string) int {
	_, year := splitId(rawEventId)
	return year
}

func splitId(rawEventId string) (string, int) {
	// Remove the letters on the left leaves us with <2 # year><id>
	yearId := strings.TrimLeftFunc(rawEventId, unicode.IsLetter)
	// Remove the numbers on the right leaves us with the event category
	category := strings.TrimRightFunc(rawEventId, unicode.IsDigit)

	twoDigitYear, err := strconv.Atoi(yearId[:2])
	if err != nil {
		log.Fatalf("Unable to parse year out of %s, %v", rawEventId, err)
	}
	if 15 > twoDigitYear {
		log.Fatalf("Unsupported year being parsed! rawEventId %s", rawEventId)
	}

	return category, 2000 + twoDigitYear
}

type SlimEvent struct {
	EventId          string
	StartTime        time.Time
	Duration         int
	EndTime          time.Time
	Location         string
	RoomName         string
	TableNumber      string
	TicketsAvailable int
	IsStarred        bool
}

type GenconEvent struct {
	EventId              string
	Year                 int
	Active               bool
	Group                string
	Title                string
	ShortDescription     string
	LongDescription      string
	EventType            string
	GameSystem           string
	RulesEdition         string
	MinPlayers           int
	MaxPlayers           int
	AgeRequired          string
	ExperienceRequired   string
	MaterialsProvided    bool
	StartTime            time.Time
	Duration             int
	EndTime              time.Time
	GMNames              string
	Website              string
	Email                string
	Tournament           bool
	RoundNumber          int
	TotalRounds          int
	MinPlayTime          int
	AttendeeRegistration string
	Cost                 int
	Location             string
	RoomName             string
	TableNumber          string
	SpecialCategory      string
	TicketsAvailable     int
	LastModified         time.Time
	ShortCategory        string
	IsStarred            bool
}

func NormalizeEvent(event *GenconEvent) *GenconEvent {
	if event.GameSystem == "Shadows of ESteren" {
		event.GameSystem = "Shadows of Esteren"
	}
	if event.GameSystem == "Dragon Age" {
		event.GameSystem = "Dragon AGE"
	}
	if event.GameSystem == "Magic: the Gathering" {
		event.GameSystem = "Magic: The Gathering"
	}
	if event.GameSystem == "7 wonders" {
		event.GameSystem = "7 Wonders"
	}
	if event.GameSystem == "Disney's Villainous" {
		event.GameSystem = "Disney Villainous"
	}

	return event
}

func (e *GenconEvent) PlannerLink() string {
	return fmt.Sprintf("http://www.genconplanner.com/event/%v", e.EventId)
}

func (e *GenconEvent) GenconLink() string {
	id := strings.TrimLeftFunc(e.EventId, unicode.IsLetter)[2:]
	return fmt.Sprintf("http://gencon.com/events/%v", id)
}

func (e *GenconEvent) SlimEvent() *SlimEvent {
	return &SlimEvent{
		EventId:          e.EventId,
		StartTime:        e.StartTime,
		Duration:         e.Duration,
		EndTime:          e.EndTime,
		Location:         e.Location,
		RoomName:         e.RoomName,
		TableNumber:      e.TableNumber,
		TicketsAvailable: e.TicketsAvailable,
		IsStarred:        e.IsStarred,
	}
}
