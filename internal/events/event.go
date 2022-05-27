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
	systemRemappings := map[string]string{
		"Shadows of ESteren":        "Shadows of Esteren",
		"Dragon Age":                "Dragon AGE",
		"Magic: the Gathering":      "Magic: The Gathering",
		"7 wonders":                 "7 Wonders",
		"Disney's Villainous":       "Disney Villainous",
		"Disney's Villianous":       "Disney Villainous",
		"Tesla vs Edison":           "Tesla vs. Edison",
		"Caverna w Forgotten Folks": "Caverna: The Cave Farmers",
		"Dead of Winter":            "Dead of Winter: A Crossroads Game",
		"Dr. Who: Blink":            "Blink!",
		"Dune Imperium":             "Dune: Imperium",
		"Firefly":                   "Firefly: The Game",
		"Formula De Mini":           "Formula Dé Mini",
		"Funkoverse":                "Funkoverse Strategy Game",
		"Marvel Villainous":         "Marvel Villainous: Infinite Power",
		"SpaceCorp: 2025-2300 AD":   "SpaceCorp: 2025-2300AD",
		"Spector Ops":               "Specter Ops",
		"Star Trek Ascendancy":      "Star Trek: Ascendancy",
		"Strat-O-matic":             "Strat-O-Matic Baseball",
		"Swords and Sorcery":        "Sword & Sorcery",
		"Dungeon Fun":               "Dungeon Party",
		"A Sonf of Ice and Fire - Miniatures Game": "A Song of Ice and Fire - Miniatures Game",
		"AEGIS Combining Robots":                   "A.E.G.I.S. Combining Robots: Season 2",
		"Anna's Roundtable":                        "Anna's Roundtable: The Fire Emblem Board Game",
		"Ascension: Tactics":                       "Ascension Tactics: Miniatures Deckbuilding Game",
		"Ascension; Tactics":                       "Ascension Tactics: Miniatures Deckbuilding Game",
		"Boss Monster":                             "Boss Monster: The Dungeon Building Card Game",
		"Captain is Dead":                          "The Captain Is Dead",
		"Carcassone":                               "Carcassonne",
		"cartagena":                                "Cartagena",
		"Clank!":                                   "Clank!: A Deck-Building Adventure",
		"Codenames Duet":                           "Codenames: Duet",
		"Cartographers: A Roll Player Tale":        "Cartographers",
		"Wrath of the Lich King":                   "World of Warcraft: Wrath of the Lich King",

		"Manhattan Project: Energy Empire":          "The Manhattan Project: Energy Empire",
		"Extraordinary Adventures: Pirates!":        "Extraordinary Adventures: Pirates",
		"Fangs: Werewolves vs. Vampires vs. Humans": "Fangs: Werewolves vs Vampires vs Humans",

		// The difference on this one is the emdash!
		"Disney: The Haunted Mansion - Call of the Spirits Game": "Disney: The Haunted Mansion – Call of the Spirits Game",
	}

	if canonicalSystem, found := systemRemappings[event.GameSystem]; found {
		event.GameSystem = canonicalSystem
	}

	if event.GameSystem == "Dominion" && event.RulesEdition == "Intrigue" {
		event.GameSystem = "Dominion: Intrigue"
	}

	if event.GameSystem == "Dungeons & Dragons Adventure Board Game" && event.RulesEdition == "Castle Ravenloft" {
		event.GameSystem = "Dungeons & Dragons: Castle Ravenloft Board Game"
	}

	if event.GameSystem == "Dungeons & Dragons Adventure Board Game" && event.RulesEdition == "The Legend of Drizzt" {
		event.GameSystem = "Dungeons & Dragons: The Legend of Drizzt Board Game"
	}

	if event.Title == "Exit: The Forgotten Island" && event.GameSystem == "EXIT" {
		event.GameSystem = "Exit: The Game – The Forgotten Island"
	}
	if event.Title == "Exit: The Haunted Rollercoaster" && event.GameSystem == "EXIT" {
		event.GameSystem = "Exit: The Game – The Haunted Roller Coaster"
	}
	if event.GameSystem == "Game of Thrones" &&
		event.Title == "Game of Thrones: The Board Game" &&
		event.RulesEdition == "2nd" {
		event.GameSystem = "A Game of Thrones: The Board Game (Second Edition)"
	}
	if event.GameSystem == "St Petersburg" {
		if event.RulesEdition == "1st" {
			event.GameSystem = "Saint Petersburg"
		} else {
			event.GameSystem = "Saint Petersburg (Second Edition)"
		}
	}

	if strings.Contains(event.Title, "The Boys: This Is Going to Hurt") && event.GameSystem == "Tabletop" {
		event.GameSystem = "The Boys: This Is Going to Hurt"
	}

	if event.GameSystem == "Sword & Sorcery" && event.RulesEdition == "Ancient Chronicles" {
		event.GameSystem = "Sword & Sorcery: Ancient Chronicles"
	}

	if event.GameSystem == "Atlantis Rising" && event.RulesEdition == "2nd" {
		event.GameSystem = "Atlantis Rising (Second Edition)"
	}

	return event
}

func (e *GenconEvent) PlannerLink() string {
	return fmt.Sprintf("/event/%v", e.EventId)
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
