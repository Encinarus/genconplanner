package events

import (
	"archive/zip"
	"bytes"
	"time"
	"io/ioutil"
	"encoding/xml"
)

type Cell struct {
	Type string `xml:"t,attr"`
	CellId string `xml:"r"`
	String string `xml:"is>t"`
	Number float64  `xml:"v"`
}

type Row struct {
	Cells []Cell `xml:"c"`
}

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
	LastModified float64
}

func parseTime(dateString string) time.Time {
	// 07/30/2015 03:00 PM
	// Mon Jan 2 15:04:05 -0700 MST 2006
	// 01/02/2006 03:04 PM
	location, _ := time.LoadLocation("America/Indianapolis")
	parsed, _ := time.ParseInLocation(
		"01/02/2006 03:04 PM",
		dateString,
		location)
	return parsed
}

func RowToEvent(row *Row) *GenconEvent {
	cells := row.Cells
	// We don't trust the end time supplied in the sheet, it's disagreed
	// with what gencon.com listed
	startTime := parseTime(cells[13].String)
	duration := (time.Duration)(1e9 * 60 * 60 * cells[14].Number)
	endTime := startTime.Add(duration)
	return &GenconEvent{
		EventId : cells[0].String,
		Group: cells[1].String,
		Title: cells[2].String,
		ShortDescription: cells[3].String,
		LongDescription: cells[4].String,
		EventType: cells[5].String,
		GameSystem: cells[6].String,
		RulesEdition: cells[7].String,
		MinPlayers: (int)(cells[8].Number),
		MaxPlayers: (int)(cells[9].Number),
		AgeRequired: cells[10].String,
		ExperienceRequired: cells[11].String,
		MaterialsProvided: cells[12].String == "Yes",
		StartTime: startTime,
		Duration: duration,
		EndTime: endTime,
		GMNames: cells[16].String,
		Website: cells[17].String,
		Email: cells[18].String,
		Tournament: cells[19].String == "Yes",
		RoundNumber: (int)(cells[20].Number),
		TotalRounds: (int)(cells[21].Number),
		MinPlayTime: (int)(cells[22].Number), // lookup how this is used
		AttendeeRegistration: cells[23].String,
		Cost: (int)(cells[24].Number),
		Location: cells[25].String,
		RoomName: cells[26].String,
		TableNumber: cells[27].String,
		SpecialCategory: cells[28].String,
		TicketsAvailable: (int)(cells[29].Number),
		LastModified: cells[30].Number,
	}
}

func ParseGenconSheet(raw_bytes []byte) []*GenconEvent {
	zipReader, err := zip.NewReader(bytes.NewReader(raw_bytes), (int64)(len(raw_bytes)))
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
	for token, err := decoder.Token(); err == nil ; token, err = decoder.Token() {
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "row"{
				if !seenHeader {
					seenHeader = true
					continue
				}
				var row Row
				decoder.DecodeElement(&row, &t)
				events = append(events, RowToEvent(&row))
			}
		}
	}
	return events
}