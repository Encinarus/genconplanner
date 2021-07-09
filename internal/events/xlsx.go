package events

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"strconv"
	"time"
)

type excelCell struct {
	Type   string  `xml:"t,attr"`
	CellId string  `xml:"r,attr"`
	String string  `xml:"is>t"`
	Number float64 `xml:"v"`
}

type excelRow struct {
	Cells []excelCell `xml:"c"`
}

func parseTime(dateString string) time.Time {
	// source format:			07/30/2015 03:00 PM
	// canonical go time: 		Mon Jan 2 15:04:05 -0700 MST 2006
	// reformated canonical: 	01/02/2006 03:04 PM
	location, _ := time.LoadLocation("America/Indianapolis")
	parsed, _ := time.ParseInLocation(
		"01/02/2006 03:04 PM",
		dateString,
		location)
	return parsed
}

func normalizeEvent(event *GenconEvent) *GenconEvent {
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

	return event
}

func rowToEvent(row *excelRow) *GenconEvent {
	cells := row.Cells
	startTime := parseTime(cells[13].String)
	duration := (int)(60 * cells[14].Number)
	// We don't trust the end time supplied in the sheet, it's disagreed
	// with what gencon.com listed, so calculate based on duration
	// time.Duration is in nano seconds, convert minutes to seconds
	endTime := startTime.Add((time.Duration)(1e9 * 60 * duration))

	eventId := cells[0].String
	shortCategory, year := splitId(eventId)

	indy, _ := time.LoadLocation("America/Indianapolis")
	excelReferenceDate := time.Date(1900, time.January, 01, 0, 0, 0, 0, indy)
	// This doesn't quite get us the last update time, but it's close enough
	lastModifiedDuration := (time.Duration)(cells[30].Number * (float64)(time.Hour) * 24)
	lastModified := excelReferenceDate.Add(lastModifiedDuration)

	title := cells[2].String
	if title == "" && cells[2].Number != 0 {
		title = strconv.FormatInt((int64)(cells[2].Number), 10)
	}
	return normalizeEvent(&GenconEvent{
		EventId:              eventId,
		Year:                 year,
		Active:               true,
		Group:                cells[1].String,
		Title:                title,
		ShortDescription:     cells[3].String,
		LongDescription:      cells[4].String,
		EventType:            cells[5].String,
		GameSystem:           cells[6].String,
		RulesEdition:         cells[7].String,
		MinPlayers:           (int)(cells[8].Number),
		MaxPlayers:           (int)(cells[9].Number),
		AgeRequired:          cells[10].String,
		ExperienceRequired:   cells[11].String,
		MaterialsProvided:    cells[12].String == "Yes",
		StartTime:            startTime,
		Duration:             duration,
		EndTime:              endTime,
		GMNames:              cells[16].String,
		Website:              cells[17].String,
		Email:                cells[18].String,
		Tournament:           cells[19].String == "Yes",
		RoundNumber:          (int)(cells[20].Number),
		TotalRounds:          (int)(cells[21].Number),
		MinPlayTime:          (int)(60 * cells[22].Number),
		AttendeeRegistration: cells[23].String,
		Cost:                 (int)(cells[24].Number),
		Location:             cells[25].String,
		RoomName:             cells[26].String,
		TableNumber:          cells[27].String,
		SpecialCategory:      cells[28].String,
		TicketsAvailable:     (int)(cells[29].Number),
		LastModified:         lastModified,
		ShortCategory:        shortCategory,
	})
}

func ParseGenconSheet(rawBytes []byte) []*GenconEvent {
	zipReader, err := zip.NewReader(bytes.NewReader(rawBytes), (int64)(len(rawBytes)))
	if err != nil {
		panic(err)
	}

	var sheet *zip.File
	for i := 0; i < len(zipReader.File); i++ {
		sheet = zipReader.File[i]
		if sheet.Name == "xl/worksheets/sheet1.xml" {
			break
		}
	}
	dataSheet, err := sheet.Open()
	if err != nil {
		panic(err)
	}
	sheetBytes, err := ioutil.ReadAll(dataSheet)
	if err != nil {
		panic(err)
	}
	decoder := xml.NewDecoder(bytes.NewBuffer(sheetBytes))

	seenHeader := false
	var events []*GenconEvent
	for token, err := decoder.Token(); err == nil; token, err = decoder.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "row" {
				// Header row won't fit in the text
				if !seenHeader {
					seenHeader = true
					continue
				}
				var row excelRow
				err = decoder.DecodeElement(&row, &t)
				if err != nil {
					panic(err)
				}
				events = append(events, rowToEvent(&row))
			}
		}
	}
	return events
}
