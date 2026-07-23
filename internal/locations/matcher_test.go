package locations

import (
	"strings"
	"testing"
)

const sampleCSVData = `id,searchable_name,location_label,map_location,category,convention_id
18522,"Field : Able Kompanie, Events",Field : Able Kompanie,/map?c=27&f=0&lg=24.628&lt=73.610&s=18522&z=5,Events,27
11636,"Hyatt : Concept A",,/map?c=0&f=2&lg=-26.806&lt=-65.183&s=11636&z=5,Spaces,0
1237,"JW Marriott : Rm 208",,/map?c=0&f=2&lg=150.12&lt=-55.12&s=1237&z=5,Spaces,0
11855,"Union Station : Nickel Plate Alcove",,/map?c=0&f=1&lg=-62.905&lt=50.353&s=11855&z=5,Spaces,0
1306,"Union Station : Milwaukee",,/map?c=0&f=1&lg=-60.644&lt=43.510&s=1306&z=5,Spaces,0
2658,"Crowne Plaza : Grand Central A",,/map?c=0&f=1&lg=-25.839&lt=47.398&s=2658&z=5,Spaces,0
18580,"Hall B : Orange, Events",,/map?c=27&f=1&lg=86.076&lt=-14.449&s=18580&z=5,Events,27
`

func TestMatcher(t *testing.T) {
	matcher := NewMatcher()
	err := matcher.LoadFromReader(strings.NewReader(sampleCSVData))
	if err != nil {
		t.Fatalf("Failed to load sample CSV reader: %v", err)
	}

	tests := []struct {
		location    string
		roomName    string
		tableNumber string
		wantSubstr  string
	}{
		{
			location:    "Stadium",
			roomName:    "Field : Able Kompanie",
			tableNumber: "HQ",
			wantSubstr:  "s=18522",
		},
		{
			location:    "ICC",
			roomName:    "Hall B : Orange",
			tableNumber: "10--15",
			wantSubstr:  "s=18580",
		},
		{
			location:    "Hyatt",
			roomName:    "Concept A",
			tableNumber: "2",
			wantSubstr:  "gencon.com/map",
		},
		{
			location:    "JW",
			roomName:    "208",
			tableNumber: "1--2",
			wantSubstr:  "s=1237",
		},
		{
			location:    "Union Station",
			roomName:    "Nickel Plate Alcove",
			tableNumber: "",
			wantSubstr:  "s=11855",
		},
		{
			location:    "Union Station",
			roomName:    "Milwaukee Alcove",
			tableNumber: "1--3",
			wantSubstr:  "s=1306",
		},
		{
			location:    "Crowne Plaza",
			roomName:    "Grand Central A--D",
			tableNumber: "",
			wantSubstr:  "gencon.com/map",
		},
		{
			location:    "416 Wabash",
			roomName:    "416 E Wabash St",
			tableNumber: "",
			wantSubstr:  "google.com/maps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.location+"_"+tt.roomName, func(t *testing.T) {
			got := matcher.MatchLocation(tt.location, tt.roomName, tt.tableNumber)
			if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("MatchLocation(%q, %q, %q) = %q, want substring %q", tt.location, tt.roomName, tt.tableNumber, got, tt.wantSubstr)
			}
		})
	}
}
