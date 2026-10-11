package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.142: WordCount mirrors wp.utils.WordCounter / post-time-to-read word count.
func TestWordCountWordsP2142(t *testing.T) {
	got := wp.WordCount("Hello world from GoPress", wp.WordCountTypeWords)
	if got != 4 {
		t.Fatalf("words: got %d want 4", got)
	}
	got = wp.WordCount("<p>Hello <strong>world</strong></p>", wp.WordCountTypeWords)
	if got != 2 {
		t.Fatalf("html words: got %d want 2", got)
	}
	got = wp.WordCount("one--two—three", wp.WordCountTypeWords)
	if got != 3 {
		t.Fatalf("connectors: got %d want 3", got)
	}
	if wp.WordCount("", wp.WordCountTypeWords) != 0 {
		t.Fatal("empty")
	}
	if wp.WordCount("   ", "syllables") != 0 {
		t.Fatal("whitespace + invalid type")
	}
}

func TestWordCountCharactersP2142(t *testing.T) {
	text := "Hi there"
	ex := wp.WordCount(text, wp.WordCountTypeCharactersExcludingSpaces)
	if ex != 7 { // H,i,t,h,e,r,e
		t.Fatalf("excluding spaces: got %d want 7", ex)
	}
	incl := wp.WordCount(text, wp.WordCountTypeCharactersIncludingSpaces)
	if incl != 8 { // + space
		t.Fatalf("including spaces: got %d want 8", incl)
	}
	ent := wp.WordCount("A&amp;B", wp.WordCountTypeCharactersExcludingSpaces)
	if ent != 3 { // A, a(entity→a), B
		t.Fatalf("entity chars: got %d want 3", ent)
	}
}

func TestWPWordCountUsesLocaleTypeP2142(t *testing.T) {
	c := &wp.Context{Locale: &wp.Locale{WordCountType: wp.WordCountTypeCharactersExcludingSpaces}}
	got := wp.WPWordCount(c, "Hi there", "")
	if got != 7 {
		t.Fatalf("locale type: got %d want 7", got)
	}
	got = wp.WPWordCount(nil, "one two", "")
	if got != 2 {
		t.Fatalf("nil context words: got %d want 2", got)
	}
	got = wp.WPWordCount(c, "one two three", wp.WordCountTypeWords)
	if got != 3 {
		t.Fatalf("explicit type overrides locale: got %d", got)
	}
}
