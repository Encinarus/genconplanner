package events

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"
)

func intField(fieldValue string, defaultValue int, fieldName string) int {
	if len(fieldValue) == 0 {
		return defaultValue
	}
	value, err := strconv.Atoi(fieldValue)
	if err != nil {
		log.Fatalf("Unable to parse field '%s' with value '%s', err %v",
			fieldName, fieldValue, err)
	}
	return value
}

func floatField(fieldValue string, defaultValue float64, fieldName string) float64 {
	if len(fieldValue) == 0 {
		return defaultValue
	}
	value, err := strconv.ParseFloat(fieldValue, 64)
	if err != nil {
		log.Fatalf("Unable to parse field '%s' with value '%s', err %v",
			fieldName, fieldValue, err)
	}
	return value
}

func linetoEvent(row []string) *GenconEvent {
	startTime := parseTime(row[13])
	duration := (int)(60 * floatField(row[14], 0, "Duration"))
	endTime := startTime.Add((time.Duration)(1e9 * 60 * duration))

	eventId := row[0]
	shortCategory, year, _, _ := splitId(eventId)

	indy, _ := time.LoadLocation("America/Indianapolis")
	lastModified, _ := time.ParseInLocation("01-02-06", row[30], indy)

	return &GenconEvent{
		EventId:              eventId,
		Year:                 year,
		Active:               true,
		Group:                row[1],
		Title:                row[2],
		ShortDescription:     row[3],
		LongDescription:      row[4],
		EventType:            row[5],
		GameSystem:           row[6],
		RulesEdition:         row[7],
		MinPlayers:           intField(row[8], 0, "MinPlayers"),
		MaxPlayers:           intField(row[9], 0, "MaxPlayers"),
		AgeRequired:          row[10],
		ExperienceRequired:   row[11],
		MaterialsProvided:    row[12] == "Yes",
		StartTime:            startTime,
		Duration:             duration,
		EndTime:              endTime,
		GMNames:              row[16],
		Website:              row[17],
		Email:                row[18],
		Tournament:           row[19] == "Yes",
		RoundNumber:          intField(row[20], 0, "RoundNumber"),
		TotalRounds:          intField(row[21], 0, "TotalRounds"),
		MinPlayTime:          (int)(60 * floatField(row[22], 0, "MinPlayTime")),
		AttendeeRegistration: row[23],
		Cost:                 (int)(floatField(row[24], 0, "Cost")),
		Location:             row[25],
		RoomName:             row[26],
		TableNumber:          row[27],
		SpecialCategory:      row[28],
		TicketsAvailable:     intField(row[29], 0, "TicketsAvailable"),
		LastModified:         lastModified,
		ShortCategory:        shortCategory,
	}
}

func ParseGenconCsv(rawBytes []byte) []*GenconEvent {
	csvReader := csv.NewReader(bytes.NewBuffer(rawBytes))
	line, err := csvReader.Read()
	if err != nil {
		fmt.Println(line)
	}
	var events = make([]*GenconEvent, 0)
	for {
		line, err = csvReader.Read()
		if err == io.EOF {
			return events
		} else if err != nil {
			log.Fatal("Errored during parsing", err)
		}

		events = append(events, linetoEvent(line))
	}
	return events
}
