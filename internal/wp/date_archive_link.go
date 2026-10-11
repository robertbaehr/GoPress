package wp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GetYearLink builds get_year_link (P3.154).
// year ≤ 0 uses current local year. Pretty path when permalink_structure set;
// otherwise ?m=YYYY. Applies year_link filter.
func GetYearLink(c *Context, year int) string {
	year = resolveArchiveYear(c, year)
	out := ""
	if usingDatePermalinks(c) {
		out = strings.TrimRight(HomeURL(c, "", ""), "/") + fmt.Sprintf("/%04d/", year)
	} else {
		out = HomeURL(c, "?m="+strconv.Itoa(year), "")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("year_link", out, year); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

// GetMonthLink builds get_month_link (P3.154).
// year/month ≤ 0 use current local values. Applies month_link filter.
func GetMonthLink(c *Context, year, month int) string {
	year = resolveArchiveYear(c, year)
	month = resolveArchiveMonth(c, month)
	out := ""
	if usingDatePermalinks(c) {
		out = strings.TrimRight(HomeURL(c, "", ""), "/") + fmt.Sprintf("/%04d/%02d/", year, month)
	} else {
		out = HomeURL(c, fmt.Sprintf("?m=%04d%02d", year, month), "")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("month_link", out, year, month); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

// GetDayLink builds get_day_link (P3.154).
// year/month/day ≤ 0 use current local values. Applies day_link filter.
func GetDayLink(c *Context, year, month, day int) string {
	year = resolveArchiveYear(c, year)
	month = resolveArchiveMonth(c, month)
	day = resolveArchiveDay(c, day)
	out := ""
	if usingDatePermalinks(c) {
		out = strings.TrimRight(HomeURL(c, "", ""), "/") + fmt.Sprintf("/%04d/%02d/%02d/", year, month, day)
	} else {
		out = HomeURL(c, fmt.Sprintf("?m=%04d%02d%02d", year, month, day), "")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("day_link", out, year, month, day); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

func usingDatePermalinks(c *Context) bool {
	if c == nil || c.Options == nil {
		return false
	}
	return strings.TrimSpace(optionValueString(c.Options.GetOption("permalink_structure", ""))) != ""
}

func archiveNow(c *Context) time.Time {
	return time.Now().In(Timezone(c))
}

func resolveArchiveYear(c *Context, year int) int {
	if year > 0 {
		return year
	}
	return archiveNow(c).Year()
}

func resolveArchiveMonth(c *Context, month int) int {
	if month > 0 {
		return month
	}
	return int(archiveNow(c).Month())
}

func resolveArchiveDay(c *Context, day int) int {
	if day > 0 {
		return day
	}
	return archiveNow(c).Day()
}
