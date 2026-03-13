package utils

import (
	"fmt"
	"math"
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

// ParseUTCOffset parses a UTC offset string ("+0800", "-0500", "+08:00") into total minutes.
func ParseUTCOffset(offset string) int {
	if offset == "" {
		return 0
	}

	sign := 1
	s := offset
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		sign = -1
		s = s[1:]
	}

	// Remove colon if present ("+08:00" → "0800")
	s = strings.ReplaceAll(s, ":", "")

	if len(s) != 4 {
		return 0
	}

	var hours, minutes int
	_, err := fmt.Sscanf(s, "%02d%02d", &hours, &minutes)
	if err != nil {
		return 0
	}

	return sign * (hours*60 + minutes)
}

// ConvertUTCToLocal converts a UTC time string (HHMM) and its UTC day change
// to local time using the given UTC offset. Returns the local time string (HHMM)
// and the adjusted day change relative to the local flight date.
//
// Parameters:
//   - utcTime: UTC time in HHMM format (e.g., "2200")
//   - utcDayChange: day offset from UTC operating day (0 = same day, 1 = next day)
//   - utcOffset: UTC offset string (e.g., "+0800", "-0500")
//   - localDateOffset: days between local flight date and UTC baseDate (from CalculateLocalDateOffset)
//
// Examples (with localDateOffset=1, i.e., local date is 1 day ahead of UTC baseDate):
//   - STD "2200", CD=0, offset "+0800" → local "0600", localCD=0
//   - STA "0010", CA=1, offset "+0800" → local "0810", localCA=0
func ConvertUTCToLocal(utcTime string, utcDayChange int, utcOffset string, localDateOffset int) (localTime string, localDayChange int) {
	hour, minute, err := ParseTime(utcTime)
	if err != nil {
		return utcTime, utcDayChange // fallback to UTC on parse error
	}

	offsetMinutes := ParseUTCOffset(utcOffset)

	// Total minutes from UTC baseDate midnight
	totalMinutes := utcDayChange*1440 + hour*60 + minute + offsetMinutes

	// Which local day this falls on (relative to UTC baseDate)
	localDay := int(math.Floor(float64(totalMinutes) / 1440.0))

	// Day change relative to the local flight date
	localDayChange = localDay - localDateOffset

	// Extract local time-of-day
	minuteOfDay := totalMinutes - localDay*1440
	localHour := minuteOfDay / 60
	localMin := minuteOfDay % 60

	localTime = fmt.Sprintf("%02d%02d", localHour, localMin)
	return localTime, localDayChange
}

// CalculateLocalDateOffset calculates how many days the local departure date
// differs from the UTC operating day. Returns 0 (same day), 1 (next day), or -1 (previous day).
//
// Examples:
//   - STD "2325" + offset "+0800" → floor((1405+480)/1440) = 1 (next day)
//   - STD "1000" + offset "+0800" → floor((600+480)/1440) = 0 (same day)
//   - STD "0200" + offset "-0500" → floor((120-300)/1440) = -1 (previous day)
func CalculateLocalDateOffset(stdUTC string, utcOffset string) int {
	hour, minute, err := ParseTime(stdUTC)
	if err != nil {
		return 0
	}

	stdMinutes := hour*60 + minute
	offsetMinutes := ParseUTCOffset(utcOffset)

	return int(math.Floor(float64(stdMinutes+offsetMinutes) / 1440.0))
}
