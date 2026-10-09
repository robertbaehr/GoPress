package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestTranslateStubReturnsMsgid(t *testing.T) {
	l := wp.NewLocale("en_US")
	if got := l.Translate("Hello", "default"); got != "Hello" {
		t.Fatalf("Translate: %q", got)
	}
	if got := l.TranslateNoop("World", "default"); got != "World" {
		t.Fatalf("TranslateNoop: %q", got)
	}
}

func TestTranslateNilLocale(t *testing.T) {
	if got := wp.Translate(nil, "x", "default"); got != "x" {
		t.Fatalf("got %q", got)
	}
	if got := wp.TranslateNoop(nil, "y", "default"); got != "y" {
		t.Fatalf("got %q", got)
	}
}

func TestContextLocaleField(t *testing.T) {
	c := wp.NewContext(context.Background())
	l := wp.NewLocale("de_DE")
	c.Locale = l
	if c.Locale != l || c.Locale.Code != "de_DE" {
		t.Fatal("Locale not wired")
	}
	if got := c.Locale.Translate("Save", "default"); got != "Save" {
		t.Fatalf("got %q", got)
	}
}
