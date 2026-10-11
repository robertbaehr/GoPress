package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.154: pretty year/month/day archive links when permalinks enabled.
func TestGetDateArchiveLinksPrettyP3154(t *testing.T) {
	c := testBootQuery(t)
	orig := c.Options.GetOption("permalink_structure", "")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("permalink_structure", orig, "yes") })
	_, _ = c.Options.UpdateOption("permalink_structure", "/%year%/%monthnum%/%day%/%postname%/", "yes")

	year := wp.GetYearLink(c, 2024)
	if !strings.HasSuffix(strings.TrimRight(year, "/"), "/2024") {
		t.Fatalf("year=%q", year)
	}
	month := wp.GetMonthLink(c, 2024, 3)
	if !strings.Contains(month, "/2024/03/") {
		t.Fatalf("month=%q", month)
	}
	day := wp.GetDayLink(c, 2024, 3, 5)
	if !strings.Contains(day, "/2024/03/05/") {
		t.Fatalf("day=%q", day)
	}

	c.Hooks.AddFilter("year_link", func(value any, _ ...any) any {
		return "https://filtered.example/year/"
	}, 10, 2)
	if got := wp.GetYearLink(c, 2024); got != "https://filtered.example/year/" {
		t.Fatalf("filtered year=%q", got)
	}
}

// P3.154: query-arg form when permalink_structure empty.
func TestGetDateArchiveLinksPlainP3154(t *testing.T) {
	c := testBootQuery(t)
	orig := c.Options.GetOption("permalink_structure", "")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("permalink_structure", orig, "yes") })
	_, _ = c.Options.UpdateOption("permalink_structure", "", "yes")

	year := wp.GetYearLink(c, 2024)
	if !strings.Contains(year, "m=2024") {
		t.Fatalf("year=%q", year)
	}
	month := wp.GetMonthLink(c, 2024, 7)
	if !strings.Contains(month, "m=202407") {
		t.Fatalf("month=%q", month)
	}
	day := wp.GetDayLink(c, 2024, 7, 9)
	if !strings.Contains(day, "m=20240709") {
		t.Fatalf("day=%q", day)
	}
}
