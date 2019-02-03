package events

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	demoTime := "07/30/2015 03:00 PM"

	parsedTime := parseTime(demoTime)

	if parsedTime.Weekday() != time.Thursday {
		t.Errorf("Expected Thursday")
	}
	if parsedTime.Hour() != 15 {
		t.Errorf("Expected 3pm")
	}
	timezoneName, offset := parsedTime.Zone()
	// offsets are in minutes, and we expect it to be -4 for EDT
	if offset != -4 * 60 * 60{
		t.Errorf("%s is in %v offset, not -4 as expected", timezoneName, offset)
	}
}
