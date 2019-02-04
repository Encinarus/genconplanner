package events

import (
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"
)


func splitId(rawEventId string) (string, int) {
	// Remove the letters on the left leaves us with <2 # year><id>
	yearId := strings.TrimLeftFunc(rawEventId, unicode.IsLetter)
	// Remove the numbers on the right leaves us with the event category
	category := strings.TrimRightFunc(rawEventId, unicode.IsDigit)

	twoDigitYear, err := strconv.Atoi(yearId[:2])
	if err != nil {
		log.Fatalf("Unable to parse year out of %s, %v", rawEventId, err)
	}
	if 15 > twoDigitYear || 19 < twoDigitYear {
		log.Fatalf("Unsupported year being parsed! rawEventId %s", rawEventId)
	}

	return category, 2000 + twoDigitYear
}

type GenconEvent struct {
	EventId string
	Year int
	Active bool
	Group string
	Title string
	ShortDescription string
	LongDescription string
	EventType string
	GameSystem string
	RulesEdition string
	MinPlayers int
	MaxPlayers int
	AgeRequired string
	ExperienceRequired string
	MaterialsProvided bool
	StartTime time.Time
	Duration int
	EndTime time.Time
	GMNames string
	Website string
	Email string
	Tournament bool
	RoundNumber int
	TotalRounds int
	MinPlayTime int
	AttendeeRegistration string
	Cost int
	Location string
	RoomName string
	TableNumber string
	SpecialCategory string
	TicketsAvailable int
	LastModified time.Time
}