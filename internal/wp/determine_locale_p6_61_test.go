package wp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSanitizeLocaleNameP661(t *testing.T) {
	if got := wp.SanitizeLocaleName("de_DE"); got != "de_DE" {
		t.Fatalf("plain=%q", got)
	}
	if got := wp.SanitizeLocaleName("fr-FR!"); got != "fr-FR" {
		t.Fatalf("strip=%q", got)
	}
	if got := wp.SanitizeLocaleName("en_US<script>"); got != "en_USscript" {
		t.Fatalf("junk=%q", got)
	}
}

func TestGetLocaleDefaultAndFilterP661(t *testing.T) {
	c := testBootQuery(t)
	if got := wp.GetLocale(c); got != "en_US" {
		// Fixture WPLANG empty → en_US.
		t.Fatalf("default=%q", got)
	}
	c.Hooks.AddFilter("locale", func(value any, _ ...any) any {
		return "de_DE"
	}, 10, 1)
	if got := wp.GetLocale(c); got != "de_DE" {
		t.Fatalf("filtered=%q", got)
	}
}

func TestDetermineLocaleLoginQueryAndCookieP661(t *testing.T) {
	c := testBootQuery(t)
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?wp_lang=fr_FR", nil)
	if got := wp.DetermineLocale(c, req); got != "fr_FR" {
		t.Fatalf("query=%q", got)
	}
	req = httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	req.AddCookie(&http.Cookie{Name: "wp_lang", Value: "de_DE"})
	if got := wp.DetermineLocale(c, req); got != "de_DE" {
		t.Fatalf("cookie=%q", got)
	}
	req = httptest.NewRequest(http.MethodGet, "/wp-login.php?wp_lang=xx!!", nil)
	if got := wp.DetermineLocale(c, req); got != "xx" {
		t.Fatalf("sanitized=%q", got)
	}
}

func TestDetermineLocaleFiltersP661(t *testing.T) {
	c := testBootQuery(t)
	c.Hooks.AddFilter("pre_determine_locale", func(value any, _ ...any) any {
		return "ja"
	}, 10, 1)
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?wp_lang=fr_FR", nil)
	if got := wp.DetermineLocale(c, req); got != "ja" {
		t.Fatalf("pre short-circuit=%q", got)
	}

	c2 := testBootQuery(t)
	c2.Hooks.AddFilter("determine_locale", func(value any, _ ...any) any {
		return "pt_BR"
	}, 10, 1)
	req = httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	if got := wp.DetermineLocale(c2, req); got != "pt_BR" {
		t.Fatalf("determine_locale filter=%q", got)
	}
}
