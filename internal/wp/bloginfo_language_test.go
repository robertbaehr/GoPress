package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBlogInfoLanguageDefault(t *testing.T) {
	if got := wp.BlogInfoLanguage(nil); got != "en-US" {
		t.Fatalf("nil context: %q", got)
	}
	c := &wp.Context{Locale: wp.NewLocale("")}
	if got := wp.BlogInfoLanguage(c); got != "en-US" {
		t.Fatalf("empty locale: %q", got)
	}
}

func TestBlogInfoLanguageFromLocale(t *testing.T) {
	c := &wp.Context{Locale: wp.NewLocale("de_DE")}
	if got := wp.BlogInfoLanguage(c); got != "de-DE" {
		t.Fatalf("got %q", got)
	}
	c.Locale = wp.NewLocale("fr_FR")
	if got := wp.BlogInfoLanguage(c); got != "fr-FR" {
		t.Fatalf("got %q", got)
	}
}
