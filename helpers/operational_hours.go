package helpers

import (
	"encoding/json"
	"fmt"
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
	Start string `json:"start"`
	End   string `json:"end"`
}

// IsWithinOperationalHours checks if current time is within operational hours
func IsWithinOperationalHours(hoursJSON string, timezone string) (bool, error) {
	if hoursJSON == "" {
		return true, nil // Default always open
	}

	var hours OperationalHours
	err := json.Unmarshal([]byte(hoursJSON), &hours)
	if err != nil {
		return false, fmt.Errorf("failed to parse operational hours: %w", err)
	}

	// Get current time in specified timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC // Fallback to UTC
	}
	now := time.Now().In(loc)

	// Get day of week
	dayHours := getDayHours(&hours, now.Weekday())

	// Parse start and end time
	startTime, err := time.ParseInLocation("15:04", dayHours.Start, loc)
	if err != nil {
		return false, fmt.Errorf("failed to parse start time: %w", err)
	}

	endTime, err := time.ParseInLocation("15:04", dayHours.End, loc)
	if err != nil {
		return false, fmt.Errorf("failed to parse end time: %w", err)
	}

	// Set date to current date
	today := now.Format("2006-01-02")
	startTime, _ = time.ParseInLocation("2006-01-02 15:04", today+" "+dayHours.Start, loc)
	endTime, _ = time.ParseInLocation("2006-01-02 15:04", today+" "+dayHours.End, loc)

	// Check if current time is within range
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
		return DayHours{Start: "00:00", End: "23:59"}
	}
}
