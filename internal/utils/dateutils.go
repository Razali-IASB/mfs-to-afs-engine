package utils

import (
	"fmt"
	"strings"
	"time"
)

// MatchesFrequency checks if a date matches the frequency pattern
// frequency examples: "1234567" (daily), "123" (Mon-Wed), "1357" (Mon,Wed,Fri,Sun)
func MatchesFrequency(date time.Time, frequency string, startDate time.Time) bool {
	if frequency == "" || frequency == "1234567" {
		return true // Daily operation
	}

	// Get day of week (1=Monday, 7=Sunday)
	dayOfWeek := int(date.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7 // Sunday = 7
	}

	// Check if this day is in the frequency pattern
	return strings.Contains(frequency, fmt.Sprintf("%d", dayOfWeek))
}

// IsWithinValidityPeriod checks if current date is within the validity period
func IsWithinValidityPeriod(currentDate, startDate, endDate time.Time) bool {
	// Normalize dates to midnight
	current := NormalizeDate(currentDate)
	start := NormalizeDate(startDate)
	end := NormalizeDate(endDate)

	return (current.Equal(start) || current.After(start)) &&
		(current.Equal(end) || current.Before(end))
}

// NormalizeDate sets time to midnight (00:00:00)
func NormalizeDate(date time.Time) time.Time {
    d := date.UTC()
    return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

// FormatDate formats date as YYYY-MM-DD
func FormatDate(date time.Time) string {
	return date.Format("2006-01-02")
}

// CalculateExpiryDate adds days to base date for TTL
func CalculateExpiryDate(baseDate time.Time, days int) time.Time {
	expiry := baseDate.AddDate(0, 0, days)
	return NormalizeDate(expiry)
}

// GetTodayDate returns current date at midnight
func GetTodayDate() time.Time {
    myt, err := time.LoadLocation("Asia/Kuala_Lumpur")
    if err != nil {
        myt = time.FixedZone("MYT", 8*60*60)
    }
    now := time.Now().In(myt)
    return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// ParseTime parses HHMM time string to hour and minute
func ParseTime(timeStr string) (hour, minute int, err error) {
	if len(timeStr) != 4 {
		return 0, 0, fmt.Errorf("invalid time format: %s (expected HHMM)", timeStr)
	}

	_, err = fmt.Sscanf(timeStr, "%02d%02d", &hour, &minute)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse time %s: %w", timeStr, err)
	}

	return hour, minute, nil
}

// ApplyDayChange applies day offset to a base date
func ApplyDayChange(baseDate time.Time, timeStr string, dayChange int) (time.Time, error) {
	hour, minute, err := ParseTime(timeStr)
	if err != nil {
		return time.Time{}, err
	}

	result := time.Date(
		baseDate.Year(),
		baseDate.Month(),
		baseDate.Day(),
		hour,
		minute,
		0,
		0,
		baseDate.Location(),
	)

	if dayChange != 0 {
		result = result.AddDate(0, 0, dayChange)
	}

	return result, nil
}

// GenerateAFSID creates deterministic composite key for AFS record
func GenerateAFSID(flightNo string, flightDate time.Time, depStation, arrStation string, legIndex int) string {
	dateStr := FormatDate(flightDate)
	leg := fmt.Sprintf("%s-%s", depStation, arrStation)
	return fmt.Sprintf("%s_%s_LEG%d_%s", flightNo, dateStr, legIndex+1, leg)
}

// GenerateBatchID creates unique batch identifier
func GenerateBatchID(index int) string {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	return fmt.Sprintf("batch_%s_%03d", timestamp, index)
}
