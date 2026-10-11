package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.141: default word count type is words.
func TestWPGetWordCountTypeDefaultP2141(t *testing.T) {
	if got := wp.WPGetWordCountType(nil); got != wp.WordCountTypeWords {
		t.Fatalf("nil ctx: %q", got)
	}
	c := &wp.Context{}
	if got := wp.WPGetWordCountType(c); got != wp.WordCountTypeWords {
		t.Fatalf("nil locale: %q", got)
	}
	c.Locale = wp.NewLocale("en_US")
	if got := wp.WPGetWordCountType(c); got != wp.WordCountTypeWords {
		t.Fatalf("default locale: %q", got)
	}
	if got := c.Locale.GetWordCountType(); got != wp.WordCountTypeWords {
		t.Fatalf("Locale.GetWordCountType: %q", got)
	}
}

// P2.141: characters_* modes honored; invalid falls back to words.
func TestWPGetWordCountTypeModesP2141(t *testing.T) {
	l := wp.NewLocale("ja")
	l.WordCountType = wp.WordCountTypeCharactersExcludingSpaces
	if got := l.GetWordCountType(); got != wp.WordCountTypeCharactersExcludingSpaces {
		t.Fatalf("excluding: %q", got)
	}
	l.WordCountType = wp.WordCountTypeCharactersIncludingSpaces
	if got := l.GetWordCountType(); got != wp.WordCountTypeCharactersIncludingSpaces {
		t.Fatalf("including: %q", got)
	}
	l.WordCountType = "syllables"
	if got := l.GetWordCountType(); got != wp.WordCountTypeWords {
		t.Fatalf("invalid fallback: %q", got)
	}
	c := &wp.Context{Locale: l}
	l.WordCountType = wp.WordCountTypeCharactersExcludingSpaces
	if got := wp.WPGetWordCountType(c); got != wp.WordCountTypeCharactersExcludingSpaces {
		t.Fatalf("ctx helper: %q", got)
	}
}
