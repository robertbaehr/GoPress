package wp_test

import (
	"regexp"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSpacesRegexpDefault(t *testing.T) {
	got := wp.SpacesRegexp(nil)
	if got != wp.DefaultSpacesRegexp {
		t.Fatalf("SpacesRegexp(nil)=%q want %q", got, wp.DefaultSpacesRegexp)
	}
	re := regexp.MustCompile(got)
	for _, s := range []string{" ", "\t", "\n", "\r", "\xC2\xA0", "&nbsp;"} {
		if !re.MatchString(s) {
			t.Fatalf("default pattern should match %q", s)
		}
	}
	if re.MatchString("x") {
		t.Fatal("default pattern should not match letter")
	}
}

func TestSpacesRegexpFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_spaces_regexp", func(value any, _ ...any) any {
		return `[ ]`
	}, 10, 1)
	got := wp.SpacesRegexp(c)
	if got != `[ ]` {
		t.Fatalf("filtered SpacesRegexp=%q", got)
	}
	re := wp.SpacesRegexpCompiled(c)
	if !re.MatchString(" ") || re.MatchString("\t") {
		t.Fatalf("compiled filtered pattern mismatch")
	}
}

func TestMatchSpaces(t *testing.T) {
	if !wp.MatchSpaces(nil, " hello") {
		t.Fatal("expected leading space")
	}
	if !wp.MatchSpaces(nil, "&nbsp;x") {
		t.Fatal("expected leading nbsp entity")
	}
	if wp.MatchSpaces(nil, "x ") {
		t.Fatal("should require match at start")
	}
}
