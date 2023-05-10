package events

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var eventCategoryRegex = regexp.MustCompile(`([A-Z]*)(\d\d)([A-Z][A-Z])(\d+)`)

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
	category, _, _, _ := splitId(rawEventId)
	return category
}

func YearFromEvent(rawEventId string) int {
	_, year, _, _ := splitId(rawEventId)
	return year
}

func splitId(rawEventId string) (string, int, string, string) {
	category := ""
	rawYear := ""
	locale := ""
	rawId := ""
	if eventCategoryRegex.MatchString(rawEventId) {
		// In 2023, gencon changed up the format of their ids. Boo.
		parsedFields := eventCategoryRegex.FindAllStringSubmatch(rawEventId, -1)
		category = parsedFields[0][1]
		rawYear = parsedFields[0][2]
		locale = parsedFields[0][3] // I assume it's locale at least
		rawId = parsedFields[0][4]
	} else {
		// This was the event id format before 2023
		// Remove the letters on the left leaves us with <2 # year><id>
		yearId := strings.TrimLeftFunc(rawEventId, unicode.IsLetter)
		rawYear = yearId[:2]
		rawId = yearId[2:]
		// Remove the numbers on the right leaves us with the event category
		category = strings.TrimRightFunc(rawEventId, unicode.IsDigit)
	}

	twoDigitYear, err := strconv.Atoi(rawYear)
	if err != nil {
		log.Fatalf("Unable to parse year out of %s, %v", rawEventId, err)
	}
	if 15 > twoDigitYear {
		log.Fatalf("Unsupported year being parsed! rawEventId %s", rawEventId)
	}

	return category, 2000 + twoDigitYear, locale, rawId
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
	OrgId            	 int64
}

func NormalizeEvent(event *GenconEvent) *GenconEvent {
	systemRemappings := map[string]string{
		"5 Year Mission": "Star Trek: Five-Year Mission",
		"7 wonders":      "7 Wonders",
		"A Sonf of Ice and Fire - Miniatures Game": "A Song of Ice and Fire - Miniatures Game",
		"AEGIS Combining Robots":                   "A.E.G.I.S. Combining Robots: Season 2",
		"Affliction":                               "AFFLICTION: Salem 1692",
		"Agatha Christie: Death in the Cards":      "Agatha Christie: Death on the Cards",
		"Age of Mythology":                         "Age of Mythology: The Boardgame",
		"Alien - Fate of the Nostromo":             "ALIEN: Fate of the Nostromo",
		"Anna's Roundtable":                        "Anna's Roundtable: The Fan Made Fire Emblem Board Game",
		"Ascension Tactics":                        "Ascension Tactics: Miniatures Deckbuilding Game",
		"Ascension":                                "Ascension: Deckbuilding Game",
		"Ascension: Tactics":                       "Ascension Tactics: Miniatures Deckbuilding Game",
		"Ascension; Tactics":                       "Ascension Tactics: Miniatures Deckbuilding Game",
		"Axis & Allies 1942":                       "Axis & Allies: 1942",
		"B-17 Queen of the Skies":                  "B-17: Queen of the Skies",
		"Battle for Greyport":                      "The Red Dragon Inn: Battle for Greyport",
		"Boss Monster":                             "Boss Monster: The Dungeon Building Card Game",
		"Broadsides and Boarding Parties":          "Broadsides & Boarding Parties",
		"Broken and Beautiful":                     "Broken and Beautiful: A Game About Kintsugi",
		"Cache Me If You Can!":                     "Cache Me If You Can!: The Geocaching Board Game",
		"Captain is Dead":                          "The Captain Is Dead",
		"Carcassone":                               "Carcassonne",
		"Cartographers: A Roll Player Tale":        "Cartographers",
		"Cartographers: Heroes":                    "Cartographers Heroes",
		"Castle Ravenloft":                         "Dungeons & Dragons: Castle Ravenloft Board Game",
		"Caverna w Forgotten Folks":                "Caverna: The Cave Farmers",
		"Caverna":                                  "Caverna: The Cave Farmers",
		"Clank!":                                   "Clank!: A Deck-Building Adventure",
		"Codenames Duet":                           "Codenames: Duet",
		"Conan by Monolith":                        "Conan",
		"Conquest Princess":                        "Conquest Princess: Fashion Is Power",
		"Dead of Winter":                           "Dead of Winter: A Crossroads Game",
		"Decorum":                                  "Décorum",
		"Destination Neptune":                      "Destination: Neptune",
		"Disney Sorcerer's Arena: Epic Alliances":  "Disney Sorcerer's Arena: Epic Alliances Core Set",
		"Disney's Villainous":                      "Disney Villainous",
		"Disney's Villianous":                      "Disney Villainous",
		"Disney: The Haunted Mansion - Call of the Spirits Game": "Disney: The Haunted Mansion – Call of the Spirits Game", // The difference on this one is the emdash!
		"Downfall of Pompeii":          "The Downfall of Pompeii",
		"Dr. Who: Blink":               "Blink!",
		"Dragon Age":                   "Dragon AGE",
		"Dragon Prince: Battlecharged": "The Dragon Prince: Battlecharged",
		"Dune Imperium":                "Dune: Imperium",
		"Dungeon Fun":                  "Dungeon Party",
		"Dungeons & Dragons: The Yawning Portal Board Game": "Dungeons & Dragons: The Yawning Portal",
		"E.T.I. Estimated Time to Invasion":                 "E.T.I.: Estimated Time to Invasion",
		"Extraordinary Adventures: Pirates!":                "Extraordinary Adventures: Pirates",
		"Fangs: Werewolves vs. Vampires vs. Humans":         "Fangs: Werewolves vs Vampires vs Humans",
		"Firefly":                          "Firefly: The Game",
		"Formula De Mini":                  "Formula Dé Mini",
		"Funkoverse":                       "Funkoverse Strategy Game",
		"Magic: the Gathering":             "Magic: The Gathering",
		"Manhattan Project: Energy Empire": "The Manhattan Project: Energy Empire",
		"Marvel Villainous":                "Marvel Villainous: Infinite Power",
		"Shadows of ESteren":               "Shadows of Esteren",
		"SpaceCorp: 2025-2300 AD":          "SpaceCorp: 2025-2300AD",
		"Spector Ops":                      "Specter Ops",
		"Star Trek Ascendancy":             "Star Trek: Ascendancy",
		"Strat-O-matic":                    "Strat-O-Matic Baseball",
		"Swords and Sorcery":               "Sword & Sorcery",
		"Tesla vs Edison":                  "Tesla vs. Edison",
		"Wrath of the Lich King":           "World of Warcraft: Wrath of the Lich King",
		"Zombie Survival":                  "Zombie Survival: The Board Game",
		"cartagena":                        "Cartagena",
		"Empyreal":                         "Empyreal: Spells & Steam",
		"Escape the Dark":                  "The Last of Us: Escape the Dark",
		"Faeries and Magical Creatures":    "Faeries & Magical Creatures",
		"Fateforge":                        "Fateforge: Chronicles of Kaan",
		"Genshin Tarot":                    "Genshin Tarot: The Fan Made Genshin Impact Board Game",
		"Great British Baking show":        "The Great British Baking Show Game",
		"Headless Horseman":                "Headless Horseman Board Game",
		"Hellboy The Board Game":           "Hellboy: The Board Game",
		"Hulk Smash":                       "The Incredible Hulk Smash",
		"Ierusalem":                        "Ierusalem: Anno Domini",
		"Kinfire Chronicles":               "Kinfire Chronicles: Night's Fall",
		"Kung-Fu Zoo":                      "Kung Fu Zoo",
		"Kutna Hora":                       "Kutná Hora: The City of Silver",
		"Ladies and Gentlmen":              "Ladies & Gentlemen",
		"Last Night on Earth":              "Last Night on Earth: The Zombie Game",
		"Legacy's Allure":                  "Legacy's Allure: Season 1",
		"Life of the Amazoia":              "Life of the Amazonia",
		"Masters of Orion":                 "Master of Orion: The Board Game",
		"My Little Pony Adventures in Equestria Deck-Building Game": "My Little Pony: Adventures in Equestria Deck-Building Game",
		"Oath":                         "Oath: Chronicles of Empire and Exile",
		"Oltree":                       "Oltréé",
		"Orleans":                      "Orléans",
		"Overboss":                     "Overboss: A Boss Monster Adventure",
		"Persona 5 Royal":              "Trick Gear: Persona 5 The Royal",
		"Planted":                      "Planted: A Game of Nature & Nurture",
		"Red Dragon Inn":               "The Red Dragon Inn",
		"Roll Camera":                  "Roll Camera!: The Filmmaking Board Game",
		"Roll to the Top":              "Roll to the Top!",
		"SHOBU":                        "SHŌBU",
		"Settlers of America":          "Catan Histories: Settlers of America – Trails to Rails",
		"Settlers of Catan":            "The Settlers of Catan",
		"Shadowgate the Living Castle": "Shadowgate: The Living Castle",
		"Smash Up: Disney Style!":      "Smash Up: Disney Edition",
		"Snow White Gemstone Mining":   "Snow White and the Seven Dwarfs: A Gemstone Mining Game",
		"Star Trek Ascendency":         "Star Trek: Ascendancy",
		"Stupid Death":                 "Stupid Deaths",
		"Sushi Go Party":               "Sushi Go Party!",
		"Suspects: Adele & Neville, Investigative Reporters": "Suspects: Adele and Neville, Investigative Reporters",
		"The Binding Of Isaac Four Souls":                    "The Binding of Isaac: Four Souls",
		"Trekking":                                           "Trekking the World",
		"Trogdor!!":                                          "Trogdor!!: The Board Game",
		"Tzolk'in":                                           "Tzolk'in: The Mayan Calendar",
		"Unmatched":                                          "Unmatched Game System",
		"Uproot Arboreal Battleship":                         "Uproot: Arboreal Battleship",
		"Villainous":                                         "Disney Villainous",
		"Viticulture World":                                  "Viticulture World: Cooperative Expansion",
		"Way Too Many Cats":                                  "Way Too Many Cats!",
		"World of Ulos":                                      "Dawn of Ulos",
		"Wrath of Ashardalon":                                "Dungeons & Dragons: Wrath of Ashardalon Board Game",
	}

	if canonicalSystem, found := systemRemappings[event.GameSystem]; found {
		event.GameSystem = canonicalSystem
	}

	if event.GameSystem == "1st" && event.Title == "Spring and Autumn: Story of China" {
		event.GameSystem = "Spring and Autumn: Story of China"
	}

	if event.GameSystem == "Scythe: Invaders from Afar Expansion" {
		event.GameSystem = "Scythe"
		event.RulesEdition = "Invaders from Afar Expansion"
	}

	if event.GameSystem == "Roll Camera! with B-Movie expansion" {
		event.GameSystem = "Roll Camera!: The Filmmaking Board Game"
		event.RulesEdition = "Roll Camera!: The B-Movie Expansion"
	}

	if event.GameSystem == "Betrayal at House on the Hill: Widows Walk Expansion" {
		event.GameSystem = "Betrayal at House on the Hill"
		event.RulesEdition = "Widows Walk Expansion"
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
	_, _, _, id := splitId(e.EventId)
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
