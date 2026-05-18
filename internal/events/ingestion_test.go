package events_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/Encinarus/genconplanner/internal/events"
)

func TestParseGenconCsv(t *testing.T) {
	header := strings.Join([]string{
		"Game ID", "Group", "Title", "Short Description", "Long Description",
		"Event Type", "Game System", "Rules Edition", "Min Players", "Max Players",
		"Age Required", "Experience Required", "Materials Provided", "Start Time",
		"Duration", "End Time", "GM Names", "Website", "Email", "Tournament",
		"Round Number", "Total Rounds", "Min Play Time", "Attendee Registration",
		"Cost", "Location", "Room Name", "Table Number", "Special Category",
		"Tickets Available", "Last Modified",
	}, ",")

	row := strings.Join([]string{
		"BGM26ND100001", "Indie Games", "Catan Championship", "Compete in Catan", "Long description",
		"BGM - Board Game", "Catan", "5th Edition", "3", "4",
		"12+", "None", "Yes", "07/30/2026 10:00 AM",
		"2", "07/30/2026 12:00 PM", "GM Alice", "http://catan.com", "alice@catan.com", "Yes",
		"1", "3", "2", "No",
		"10", "ICC", "Hall A", "Table 1", "",
		"15", "07-30-26",
	}, ",")

	csvData := fmt.Sprintf("%s\n%s\n", header, row)

	parsedEvents := events.ParseGenconCsv([]byte(csvData))

	if len(parsedEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(parsedEvents))
	}

	evt := parsedEvents[0]
	if evt.EventId != "BGM26ND100001" {
		t.Errorf("expected EventId BGM26ND100001, got %s", evt.EventId)
	}
	if evt.Title != "Catan Championship" {
		t.Errorf("expected Title Catan Championship, got %s", evt.Title)
	}
	if evt.Duration != 120 { // 2 hours = 120 minutes
		t.Errorf("expected Duration 120, got %d", evt.Duration)
	}
	if evt.TicketsAvailable != 15 {
		t.Errorf("expected TicketsAvailable 15, got %d", evt.TicketsAvailable)
	}
}

func TestParseGenconSheet(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	f, err := w.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	// Create XML content representing 1 header row and 1 data row with 32 cells
	var xmlBuilder strings.Builder
	xmlBuilder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet><sheetData>`)

	// Header row
	xmlBuilder.WriteString(`<row r="1">`)
	for i := 0; i < 32; i++ {
		xmlBuilder.WriteString(fmt.Sprintf(`<c r="%c1" t="s"><is><t>Header%d</t></is></c>`, 'A'+i, i))
	}
	xmlBuilder.WriteString(`</row>`)

	// Data row
	xmlBuilder.WriteString(`<row r="2">`)
	cellValues := make([]string, 32)
	cellValues[0] = "BGM26ND100002"            // EventId
	cellValues[1] = "Indie Games"              // Group
	cellValues[2] = "Catan Tourney"            // Title
	cellValues[3] = "Short desc"               // ShortDescription
	cellValues[4] = "Long desc"                // LongDescription
	cellValues[5] = "BGM - Board Game"         // EventType
	cellValues[6] = "Catan"                    // GameSystem
	cellValues[7] = "5th Edition"              // RulesEdition
	cellValues[8] = "3"                        // MinPlayers
	cellValues[9] = "4"                        // MaxPlayers
	cellValues[10] = "12+"                     // AgeRequired
	cellValues[11] = "None"                    // ExperienceRequired
	cellValues[12] = "Yes"                     // MaterialsProvided
	cellValues[13] = "07/30/2026 10:00 AM"     // StartTime (sheet format)
	cellValues[14] = "07/30/2026 10:00 AM"     // StartTime cell in rowToEvent (index 14)
	cellValues[15] = "2"                       // Duration (index 15)
	cellValues[17] = "GM Bob"                  // GMNames
	cellValues[18] = "http://catan.com"        // Website
	cellValues[19] = "bob@catan.com"           // Email
	cellValues[20] = "Yes"                     // Tournament
	cellValues[21] = "1"                       // RoundNumber
	cellValues[22] = "3"                       // TotalRounds
	cellValues[23] = "2"                       // MinPlayTime
	cellValues[24] = "No"                      // AttendeeRegistration
	cellValues[25] = "10"                      // Cost
	cellValues[26] = "ICC"                     // Location
	cellValues[27] = "Hall A"                  // RoomName
	cellValues[28] = "Table 2"                 // TableNumber
	cellValues[29] = ""                        // SpecialCategory
	cellValues[30] = "20"                      // TicketsAvailable
	cellValues[31] = "45000"                   // LastModified (excel serial date)

	for i, val := range cellValues {
		// Determine cell type: if it's a number, use <v>, otherwise <is><t>
		if i == 8 || i == 9 || i == 15 || i == 21 || i == 22 || i == 23 || i == 25 || i == 30 || i == 31 {
			xmlBuilder.WriteString(fmt.Sprintf(`<c r="%c2"><v>%s</v></c>`, 'A'+i, val))
		} else {
			xmlBuilder.WriteString(fmt.Sprintf(`<c r="%c2" t="s"><is><t>%s</t></is></c>`, 'A'+i, val))
		}
	}
	xmlBuilder.WriteString(`</row></sheetData></worksheet>`)

	_, err = f.Write([]byte(xmlBuilder.String()))
	if err != nil {
		t.Fatalf("failed to write xml content: %v", err)
	}

	w.Close()

	parsedEvents := events.ParseGenconSheet(buf.Bytes())

	if len(parsedEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(parsedEvents))
	}

	evt := parsedEvents[0]
	if evt.EventId != "BGM26ND100002" {
		t.Errorf("expected EventId BGM26ND100002, got %s", evt.EventId)
	}
	if evt.Title != "Catan Tourney" {
		t.Errorf("expected Title Catan Tourney, got %s", evt.Title)
	}
	if evt.Duration != 120 { // 2 hours = 120 minutes
		t.Errorf("expected Duration 120, got %d", evt.Duration)
	}
	if evt.TicketsAvailable != 20 {
		t.Errorf("expected TicketsAvailable 20, got %d", evt.TicketsAvailable)
	}
}
