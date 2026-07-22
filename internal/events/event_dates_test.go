package events

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestGenconDates_ConsistencyWithFrontend(t *testing.T) {
	tsPath := filepath.Join("..", "..", "ui", "src", "app", "constants", "gencon-dates.ts")
	if _, err := os.Stat(tsPath); os.IsNotExist(err) {
		t.Skipf("Skipping frontend date consistency check: %s not present (e.g. inside Docker build environment)", tsPath)
	}

	content, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("Failed to read frontend gencon-dates.ts: %v", err)
	}

	// Match lines like: 2026: { startDate: '2026-07-29', endDate: '2026-08-02' }
	re := regexp.MustCompile(`(\d{4}):\s*\{\s*startDate:\s*'([^']+)'\s*,\s*endDate:\s*'([^']+)'\s*\}`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	if len(matches) == 0 {
		t.Fatalf("No date entries parsed from frontend gencon-dates.ts")
	}

	frontendDates := make(map[int][]string)
	for _, m := range matches {
		year, _ := strconv.Atoi(m[1])
		frontendDates[year] = []string{m[2], m[3]}
	}

	// Compare Go backend genconDates with TypeScript frontend dates
	for year, goDates := range genconDates {
		tsDates, found := frontendDates[year]
		if !found {
			t.Errorf("Year %d exists in backend event.go but missing in frontend gencon-dates.ts", year)
			continue
		}
		if goDates[0] != tsDates[0] || goDates[1] != tsDates[1] {
			t.Errorf("Date mismatch for year %d: Go has %v, TS has %v", year, goDates, tsDates)
		}
	}

	for year, tsDates := range frontendDates {
		goDates, found := genconDates[year]
		if !found {
			t.Errorf("Year %d exists in frontend gencon-dates.ts but missing in backend event.go", year)
			continue
		}
		if goDates[0] != tsDates[0] || goDates[1] != tsDates[1] {
			t.Errorf("Date mismatch for year %d: TS has %v, Go has %v", year, tsDates, goDates)
		}
	}
}
