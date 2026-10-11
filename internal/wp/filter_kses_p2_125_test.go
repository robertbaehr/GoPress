package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFilterPostKsesSlashRoundTrip(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	// Slashed input with a disallowed script tag.
	in := wp.SlashString(`<p>Hi</p><script>alert(1)</script>`)
	out := wp.FilterPostKses(c, in)
	if !strings.Contains(out, `<p>Hi</p>`) {
		t.Fatalf("expected paragraph kept: %q", out)
	}
	if strings.Contains(out, "<script") || strings.Contains(strings.ToLower(out), "alert") {
		t.Fatalf("script should be stripped: %q", out)
	}
	// Result is slashed — apostrophe/backslash escapes present when needed.
	if out != wp.SlashString(wp.UnslashString(out)) {
		t.Fatalf("output should be slashed form: %q", out)
	}
}

func TestFilterKsesUsesCurrentFilterAllowList(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	var got string
	c.Hooks.AddFilter("title_save_pre", func(value any, _ ...any) any {
		s, _ := value.(string)
		got = wp.FilterKses(c, s)
		return got
	}, 10, 1)
	// title_save_pre → current_filter allow list (defaults toward post/data via AllowedHTMLFiltered).
	in := wp.SlashString(`<b>Bold</b><script>x</script>`)
	_ = c.Hooks.ApplyFilters("title_save_pre", in)
	if got == "" {
		t.Fatal("filter did not run")
	}
	if !strings.Contains(got, "<b>") && !strings.Contains(got, "Bold") {
		t.Fatalf("expected bold content kept: %q", got)
	}
	if strings.Contains(got, "<script") {
		t.Fatalf("script leaked: %q", got)
	}
	if c.Hooks.CurrentFilter() != "" {
		t.Fatalf("current filter should clear: %q", c.Hooks.CurrentFilter())
	}
}

func TestFilterKsesDefaultDataContext(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	// Data allow list permits <b> but not event attributes / scripts.
	in := wp.SlashString(`<b onclick="x">ok</b><script>bad</script>`)
	out := wp.FilterKses(c, in)
	un := wp.UnslashString(out)
	if strings.Contains(un, "onclick") || strings.Contains(un, "<script") {
		t.Fatalf("data context should strip events/scripts: %q", un)
	}
	if !strings.Contains(un, "<b>") || !strings.Contains(un, "ok") {
		t.Fatalf("expected b kept: %q", un)
	}
}
