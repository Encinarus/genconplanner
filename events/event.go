package events

import "time"

type GenconEvent struct {
	EventId string
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
	Duration time.Duration
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
	// This _should_ be a time.Time, but this is an excel datetime
	// which 1) is a pain to parse and 2) comparable anyway.
	LastModified float64
}