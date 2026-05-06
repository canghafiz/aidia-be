package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// OperationalHours represents business hours for each day
type OperationalHours struct {
	Monday    DayHours `json:"monday"`
	Tuesday   DayHours `json:"tuesday"`
	Wednesday DayHours `json:"wednesday"`
	Thursday  DayHours `json:"thursday"`
	Friday    DayHours `json:"friday"`
	Saturday  DayHours `json:"saturday"`
	Sunday    DayHours `json:"sunday"`
}

// DayHours represents open/close time for a single day
type DayHours struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	Closed bool   `json:"closed"`
}

// IsWithinOperationalHours checks if current time is within operational hours.
// hoursJSON must be a structured JSON object produced by OperationalHoursField.
// Returns (true, nil) when no config set. Returns (false, nil) on parse failure — fail-safe.
func IsWithinOperationalHours(hoursJSON string, timezone string) (bool, error) {
	if hoursJSON == "" {
		return true, nil // no config = always open
	}

	var hours OperationalHours
	if err := json.Unmarshal([]byte(hoursJSON), &hours); err != nil {
		// Not valid JSON (e.g. old plain-text format) — fail safe: treat as closed
		log.Printf("[OperationalHours] Cannot parse JSON: %v", err)
		return false, nil
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	dayHours := getDayHours(&hours, now.Weekday())

	// Explicitly closed today
	if dayHours.Closed {
		return false, nil
	}

	// No hours configured for this day → open all day
	if dayHours.Start == "" || dayHours.End == "" {
		return true, nil
	}

	today := now.Format("2006-01-02")
	startTime, err := time.ParseInLocation("2006-01-02 15:04", today+" "+dayHours.Start, loc)
	if err != nil {
		log.Printf("[OperationalHours] Bad start time %q: %v", dayHours.Start, err)
		return false, nil
	}
	endTime, err := time.ParseInLocation("2006-01-02 15:04", today+" "+dayHours.End, loc)
	if err != nil {
		log.Printf("[OperationalHours] Bad end time %q: %v", dayHours.End, err)
		return false, nil
	}

	return now.After(startTime) && now.Before(endTime), nil
}

// IsBotEnabled checks if bot is enabled
func IsBotEnabled(enabled string) bool {
	return enabled == "true" || enabled == ""
}

// IsManualMode checks if bot is in manual mode
func IsManualMode(manualMode string) bool {
	return manualMode == "true"
}

// FormatOperationalHoursForAI converts the JSON schedule to human-readable text for OpenAI.
// If hoursJSON is not valid JSON (old plain-text format), it is returned as-is.
func FormatOperationalHoursForAI(hoursJSON string) string {
	if hoursJSON == "" {
		return ""
	}
	var hours OperationalHours
	if err := json.Unmarshal([]byte(hoursJSON), &hours); err != nil {
		return hoursJSON // old plain-text — pass through unchanged
	}

	days := []struct {
		name string
		h    DayHours
	}{
		{"Monday", hours.Monday},
		{"Tuesday", hours.Tuesday},
		{"Wednesday", hours.Wednesday},
		{"Thursday", hours.Thursday},
		{"Friday", hours.Friday},
		{"Saturday", hours.Saturday},
		{"Sunday", hours.Sunday},
	}

	var lines []string
	for _, d := range days {
		if d.h.Closed {
			lines = append(lines, fmt.Sprintf("- %s: Closed", d.name))
		} else if d.h.Start != "" && d.h.End != "" {
			lines = append(lines, fmt.Sprintf("- %s: %s - %s", d.name, d.h.Start, d.h.End))
		} else {
			lines = append(lines, fmt.Sprintf("- %s: Open all day", d.name))
		}
	}
	return strings.Join(lines, "\n")
}

// getDayHours returns the hours for a specific day of week
func getDayHours(hours *OperationalHours, weekday time.Weekday) DayHours {
	switch weekday {
	case time.Monday:
		return hours.Monday
	case time.Tuesday:
		return hours.Tuesday
	case time.Wednesday:
		return hours.Wednesday
	case time.Thursday:
		return hours.Thursday
	case time.Friday:
		return hours.Friday
	case time.Saturday:
		return hours.Saturday
	case time.Sunday:
		return hours.Sunday
	default:
		return DayHours{}
	}
}
