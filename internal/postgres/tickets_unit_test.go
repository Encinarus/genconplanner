package postgres

import (
	"testing"
	"time"
)

func TestFormatIndianaTime(t *testing.T) {
	// 10:00 AM EDT (UTC-4) is stored in PostgreSQL as 14:00:00 UTC
	utcTime := time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC)

	formatted := formatIndianaTime(utcTime)
	expected := "2026-07-30T10:00:00-04:00"

	if formatted != expected {
		t.Errorf("expected formatIndianaTime(%v) to be %q, got %q", utcTime, expected, formatted)
	}

	// Test zero time / placeholder 1970
	zeroTime := time.Time{}
	if got := formatIndianaTime(zeroTime); got != "" {
		t.Errorf("expected empty string for zero time, got %q", got)
	}

	epochTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := formatIndianaTime(epochTime); got != "" {
		t.Errorf("expected empty string for 1970 placeholder time, got %q", got)
	}
}
