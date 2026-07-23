package locations

import (
	"context"
	"database/sql"
	"encoding/csv"
	"io"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type LocationPin struct {
	ID             int
	SearchableName string
	LocationLabel  string
	MapLocation    string
	Category       string
	ConventionID   int
}

type Matcher struct {
	mu                 sync.RWMutex
	labelIndex         map[string]*LocationPin
	searchableIndex    map[string]*LocationPin
	venueRoomIndex     map[string]*LocationPin
	subroomIndex       map[string][]*LocationPin
	buildingVenueIndex map[string]*LocationPin
}

func NewMatcher() *Matcher {
	return &Matcher{
		labelIndex:         make(map[string]*LocationPin),
		searchableIndex:    make(map[string]*LocationPin),
		venueRoomIndex:     make(map[string]*LocationPin),
		subroomIndex:       make(map[string][]*LocationPin),
		buildingVenueIndex: make(map[string]*LocationPin),
	}
}

// LoadFromDB loads location pins from database table public.gencon_locations
func (m *Matcher) LoadFromDB(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "SELECT id, searchable_name, COALESCE(location_label, ''), map_location, category, convention_id FROM public.gencon_locations")
	if err != nil {
		return err
	}
	defer rows.Close()

	var pins []*LocationPin
	for rows.Next() {
		var pin LocationPin
		if err := rows.Scan(&pin.ID, &pin.SearchableName, &pin.LocationLabel, &pin.MapLocation, &pin.Category, &pin.ConventionID); err != nil {
			return err
		}
		pins = append(pins, &pin)
	}

	m.BuildIndex(pins)
	log.Printf("Loaded %d map location pins from database.", len(pins))
	return nil
}

// LoadFromReader loads location pins from any io.Reader containing CSV data
func (m *Matcher) LoadFromReader(r io.Reader) error {
	reader := csv.NewReader(r)
	if _, err := reader.Read(); err != nil { // Skip header
		return err
	}

	var pins []*LocationPin
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(record) < 6 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		convID, _ := strconv.Atoi(record[5])

		pins = append(pins, &LocationPin{
			ID:             id,
			SearchableName: record[1],
			LocationLabel:  record[2],
			MapLocation:    record[3],
			Category:       record[4],
			ConventionID:   convID,
		})
	}

	m.BuildIndex(pins)
	return nil
}

var categorySuffixRegex = regexp.MustCompile(`(?i),\s*(Events|Exhibitors|Spaces|Concessions)\s*$`)
var romanMap = []struct{ r, a string }{
	{" viii", " 8"}, {" vii", " 7"}, {" vi", " 6"}, {" iv", " 4"},
	{" v", " 5"}, {" iii", " 3"}, {" ii", " 2"}, {" i", " 1"},
}
var roomWordsRegex = regexp.MustCompile(`(?i)\b(room|rm|bllrm|blrm|ballroom|stn|station)\b`)
var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]`)
var modifierRegex = regexp.MustCompile(`(?i)\b(alcove|foyer|antechamber|mezzanine|balcony|nook)\b`)

func cleanTokens(text string) string {
	t := strings.ToLower(strings.TrimSpace(text))
	t = roomWordsRegex.ReplaceAllString(t, "")
	if idx := strings.Index(t, "--"); idx != -1 {
		t = t[:idx]
	}
	t = nonAlphaNumRegex.ReplaceAllString(t, " ")
	return strings.Join(strings.Fields(t), " ")
}

func romanToArabic(text string) string {
	t := strings.ToLower(text)
	for _, pair := range romanMap {
		if strings.HasSuffix(t, pair.r) || strings.Contains(t, pair.r+" ") {
			t = strings.ReplaceAll(t, pair.r, pair.a)
		}
	}
	return t
}

func (m *Matcher) PinCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.searchableIndex) + len(m.labelIndex)
}

func (m *Matcher) BuildIndex(pins []*LocationPin) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.labelIndex = make(map[string]*LocationPin)
	m.searchableIndex = make(map[string]*LocationPin)
	m.venueRoomIndex = make(map[string]*LocationPin)
	m.subroomIndex = make(map[string][]*LocationPin)
	m.buildingVenueIndex = make(map[string]*LocationPin)

	validVenues := map[string]bool{
		"icc": true, "jw marriott": true, "crowne plaza": true, "westin": true,
		"hyatt": true, "stadium": true, "marriott": true, "embassy suites": true,
		"hilton": true, "omni": true, "le meridien": true,
	}

	for _, pin := range pins {
		if pin.LocationLabel != "" {
			m.labelIndex[strings.ToLower(pin.LocationLabel)] = pin
		}

		cleanS := categorySuffixRegex.ReplaceAllString(pin.SearchableName, "")
		cleanS = strings.ToLower(strings.TrimSpace(cleanS))
		if cleanS != "" {
			m.searchableIndex[cleanS] = pin
		}

		parts := strings.Split(pin.SearchableName, ":")
		ven := strings.ToLower(strings.TrimSpace(parts[0]))
		rm := ""
		if len(parts) > 1 {
			rm = strings.TrimSpace(parts[1])
		}

		cV := cleanTokens(ven)
		cR := cleanTokens(rm)

		if pin.ConventionID == 0 && cV != "" && validVenues[cV] {
			m.buildingVenueIndex[cV] = pin
		}

		if cV != "" && cR != "" {
			key := cV + "::" + cR
			m.venueRoomIndex[key] = pin
			m.subroomIndex[cV] = append(m.subroomIndex[cV], pin)
		}
	}
}

// MatchLocation attempts to find a matching map link for an event's location/room/table.
func (m *Matcher) MatchLocation(location, roomName, tableNumber string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	loc := strings.TrimSpace(location)
	room := strings.TrimSpace(roomName)

	if loc == "" && room == "" {
		return ""
	}

	// Step 1: Exact Match on Room Label (e.g. "Field : Able Kompanie") or Clean Searchable Name (e.g. "Hall B : Orange")
	if room != "" {
		if pin, found := m.labelIndex[strings.ToLower(room)]; found {
			return formatGenconMapURL(pin.MapLocation)
		}
		if pin, found := m.searchableIndex[strings.ToLower(room)]; found {
			return formatGenconMapURL(pin.MapLocation)
		}
	}

	fullQ := strings.ToLower(loc)
	if room != "" {
		fullQ = strings.ToLower(loc + " : " + room)
	}

	if pin, found := m.searchableIndex[fullQ]; found {
		return formatGenconMapURL(pin.MapLocation)
	}

	// Canonical Venue Alias Normalization
	cLoc := cleanTokens(loc)
	if cLoc == "jw" {
		cLoc = "jw marriott"
	} else if cLoc == "union station" || cLoc == "umiom station" {
		cLoc = "crowne plaza"
	}

	cRm := cleanTokens(room)
	cRmRoman := cleanTokens(romanToArabic(room))

	// Step 2: Normalized Room & Alias Match
	if cLoc != "" && cRm != "" {
		if pin, found := m.venueRoomIndex[cLoc+"::"+cRm]; found {
			return formatGenconMapURL(pin.MapLocation)
		}
	}
	if cLoc != "" && cRmRoman != "" {
		if pin, found := m.venueRoomIndex[cLoc+"::"+cRmRoman]; found {
			return formatGenconMapURL(pin.MapLocation)
		}
	}

	// Step 3: Sub-Room / Range Breakdown Match
	if cLoc != "" && cRm != "" {
		fields := strings.Fields(cRm)
		if len(fields) > 0 {
			baseRm := fields[0]
			for _, cand := range m.subroomIndex[cLoc] {
				parts := strings.Split(cand.SearchableName, ":")
				candRm := ""
				if len(parts) > 1 {
					candRm = cleanTokens(parts[1])
				}
				if strings.Contains(candRm, baseRm) {
					return formatGenconMapURL(cand.MapLocation)
				}
			}
		}
	}

	// Step 4: Modifier Stripping Fallback ("Alcove", "Foyer", etc.)
	if room != "" {
		modRm := modifierRegex.ReplaceAllString(room, "")
		cModRm := cleanTokens(modRm)
		if modRm != room && cModRm != "" {
			if pin, found := m.venueRoomIndex[cLoc+"::"+cModRm]; found {
				return formatGenconMapURL(pin.MapLocation)
			}
			fields := strings.Fields(cModRm)
			if len(fields) > 0 {
				modBase := fields[0]
				for _, cand := range m.subroomIndex[cLoc] {
					parts := strings.Split(cand.SearchableName, ":")
					candRm := ""
					if len(parts) > 1 {
						candRm = cleanTokens(parts[1])
					}
					if strings.Contains(candRm, modBase) {
						return formatGenconMapURL(cand.MapLocation)
					}
				}
			}
		}
	}

	// Step 5: Venue Building Level Marker Fallback
	if cLoc != "" {
		if pin, found := m.buildingVenueIndex[cLoc]; found {
			return formatGenconMapURL(pin.MapLocation)
		}
	}

	// Step 6: Offsite Google Maps Fallback
	query := loc
	if room != "" {
		query += " " + room
	}
	query += " Indianapolis IN"
	return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(query)
}

func formatGenconMapURL(relPath string) string {
	if relPath == "" {
		return ""
	}
	if strings.HasPrefix(relPath, "http://") || strings.HasPrefix(relPath, "https://") {
		return relPath
	}
	if !strings.HasPrefix(relPath, "/") {
		relPath = "/" + relPath
	}
	return "https://www.gencon.com" + relPath
}
