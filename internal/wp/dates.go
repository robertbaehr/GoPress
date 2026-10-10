// Date/time helpers (P2.13 / P2.27) — subset of wp-includes/functions.php date APIs.
package wp

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const mysqlDateLayout = "2006-01-02 15:04:05"

// WP time unit constants (wp-includes/default-constants.php).
const (
	MinuteInSeconds = 60
	HourInSeconds   = 3600
	DayInSeconds    = 86400
	WeekInSeconds   = 604800
	MonthInSeconds  = 2592000
	YearInSeconds   = 31536000
)

// WPDate formats a Unix timestamp in timezone tz using a PHP date() format
// subset (wp_date). When tz is nil, UTC is used.
func WPDate(format string, timestamp int64, tz *time.Location) string {
	if tz == nil {
		tz = time.UTC
	}
	t := time.Unix(timestamp, 0).In(tz)
	return formatPHPDate(format, t)
}

// DateI18n is a thin wrapper around WPDate using the named IANA timezone
// (date_i18n subset when a timezone_string is in effect).
func DateI18n(format string, timestamp int64, tzName string) (string, error) {
	loc, err := loadTZ(tzName)
	if err != nil {
		return "", err
	}
	return WPDate(format, timestamp, loc), nil
}

// MySQL2Date converts a MySQL DATETIME string to format (mysql2date subset).
// When gmt is true the value is treated as already-GMT; for the common
// identity format Y-m-d H:i:s both paths return the normalized datetime.
func MySQL2Date(format, mysqlDatetime string, gmt bool) string {
	_ = gmt
	mysqlDatetime = strings.TrimSpace(mysqlDatetime)
	if mysqlDatetime == "" || mysqlDatetime == "0000-00-00 00:00:00" {
		return ""
	}
	t, err := time.ParseInLocation(mysqlDateLayout, mysqlDatetime, time.UTC)
	if err != nil {
		return "0"
	}
	if format == "U" || format == "G" {
		return strconv.FormatInt(t.Unix(), 10)
	}
	return formatPHPDate(format, t)
}

// GetGMTFromDate converts a local MySQL DATETIME in tzName to UTC (get_gmt_from_date).
func GetGMTFromDate(local, tzName, format string) (string, error) {
	loc, err := loadTZ(tzName)
	if err != nil {
		return "", err
	}
	if format == "" {
		format = "Y-m-d H:i:s"
	}
	t, err := time.ParseInLocation(mysqlDateLayout, strings.TrimSpace(local), loc)
	if err != nil {
		return "", err
	}
	return formatPHPDate(format, t.UTC()), nil
}

// GetDateFromGMT converts a UTC MySQL DATETIME to local tzName (get_date_from_gmt).
func GetDateFromGMT(gmt, tzName, format string) (string, error) {
	loc, err := loadTZ(tzName)
	if err != nil {
		return "", err
	}
	if format == "" {
		format = "Y-m-d H:i:s"
	}
	t, err := time.ParseInLocation(mysqlDateLayout, strings.TrimSpace(gmt), time.UTC)
	if err != nil {
		return "", err
	}
	return formatPHPDate(format, t.In(loc)), nil
}

func loadTZ(name string) (*time.Location, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "UTC") {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("wp: timezone %q: %w", name, err)
	}
	return loc, nil
}

// formatPHPDate implements a small PHP date() format subset used by golden vectors.
func formatPHPDate(format string, t time.Time) string {
	switch format {
	case "U":
		return strconv.FormatInt(t.Unix(), 10)
	case "c":
		// PHP date('c') uses numeric offsets (+00:00), not Z.
		return t.Format("2006-01-02T15:04:05-07:00")
	case "Y-m-d H:i:s":
		return t.Format(mysqlDateLayout)
	case "F j, Y":
		return t.Format("January 2, 2006")
	}
	// Generic character-by-character conversion for simple formats.
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		ch := format[i]
		switch ch {
		case '\\':
			if i+1 < len(format) {
				i++
				b.WriteByte(format[i])
			}
		case 'Y':
			b.WriteString(t.Format("2006"))
		case 'm':
			b.WriteString(t.Format("01"))
		case 'd':
			b.WriteString(t.Format("02"))
		case 'H':
			b.WriteString(t.Format("15"))
		case 'i':
			b.WriteString(t.Format("04"))
		case 's':
			b.WriteString(t.Format("05"))
		case 'F':
			b.WriteString(t.Format("January"))
		case 'j':
			b.WriteString(strconv.Itoa(t.Day()))
		case 'U':
			b.WriteString(strconv.FormatInt(t.Unix(), 10))
		case 'c':
			b.WriteString(t.Format("2006-01-02T15:04:05-07:00"))
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// HumanTimeDiff returns a human-readable difference between two Unix timestamps
// (human_time_diff). When to is 0, time.Now() is used.
func HumanTimeDiff(from, to int64) string {
	if to == 0 {
		to = time.Now().Unix()
	}
	diff := int64(math.Abs(float64(to - from)))
	switch {
	case diff < MinuteInSeconds:
		secs := diff
		if secs <= 1 {
			secs = 1
		}
		return pluralDiff(secs, "second", "seconds")
	case diff < HourInSeconds:
		mins := int64(math.Round(float64(diff) / MinuteInSeconds))
		if mins <= 1 {
			mins = 1
		}
		return pluralDiff(mins, "minute", "minutes")
	case diff < DayInSeconds:
		hours := int64(math.Round(float64(diff) / HourInSeconds))
		if hours <= 1 {
			hours = 1
		}
		return pluralDiff(hours, "hour", "hours")
	case diff < WeekInSeconds:
		days := int64(math.Round(float64(diff) / DayInSeconds))
		if days <= 1 {
			days = 1
		}
		return pluralDiff(days, "day", "days")
	case diff < MonthInSeconds:
		weeks := int64(math.Round(float64(diff) / WeekInSeconds))
		if weeks <= 1 {
			weeks = 1
		}
		return pluralDiff(weeks, "week", "weeks")
	case diff < YearInSeconds:
		months := int64(math.Round(float64(diff) / MonthInSeconds))
		if months <= 1 {
			months = 1
		}
		return pluralDiff(months, "month", "months")
	default:
		years := int64(math.Round(float64(diff) / YearInSeconds))
		if years <= 1 {
			years = 1
		}
		return pluralDiff(years, "year", "years")
	}
}

func pluralDiff(n int64, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
