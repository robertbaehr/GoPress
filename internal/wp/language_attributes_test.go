package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetLanguageAttributesDefault(t *testing.T) {
	c := &wp.Context{Locale: wp.NewLocale("en_US")}
	got := wp.GetLanguageAttributes(c, "html")
	if got != `lang="en-US"` {
		t.Fatalf("got %q", got)
	}
	html := wp.LanguageAttributesHTML(c)
	if html != ` lang="en-US"` {
		t.Fatalf("html=%q", html)
	}
}

func TestGetLanguageAttributesRTLAndFilter(t *testing.T) {
	c := &wp.Context{
		Locale: wp.NewLocale("ar"),
		Hooks:  wp.NewHooks(),
	}
	got := wp.GetLanguageAttributes(c, "html")
	if !strings.Contains(got, `dir="rtl"`) || !strings.Contains(got, `lang="ar"`) {
		t.Fatalf("rtl got %q", got)
	}
	c.Hooks.AddFilter("language_attributes", func(value any, args ...any) any {
		return `lang="xx-TEST"`
	}, 10, 2)
	got = wp.GetLanguageAttributes(c, "html")
	if got != `lang="xx-TEST"` {
		t.Fatalf("filter got %q", got)
	}
}

func TestGetLanguageAttributesXHTML(t *testing.T) {
	c := &wp.Context{Locale: wp.NewLocale("de_DE")}
	got := wp.GetLanguageAttributes(c, "xhtml")
	if !strings.Contains(got, `lang="de-DE"`) || !strings.Contains(got, `xml:lang="de-DE"`) {
		t.Fatalf("xhtml got %q", got)
	}
}
