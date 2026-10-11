package login

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P6.61: language switcher selected follows determine_locale / sanitize.
func TestWriteLanguageSwitcherDetermineLocaleP661(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("determine_locale", func(value any, _ ...any) any {
		return "pt_BR"
	}, 10, 1)
	var b strings.Builder
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	writeLanguageSwitcher(&b, req, c, []string{"de_DE", "pt_BR"})
	out := b.String()
	if !strings.Contains(out, `value="pt_BR" selected="selected"`) {
		t.Fatalf("expected determine_locale selected: %q", out)
	}

	// Junk query sanitized before option match.
	var b2 strings.Builder
	req2 := httptest.NewRequest(http.MethodGet, "/wp-login.php?wp_lang=de_DE!!!", nil)
	writeLanguageSwitcher(&b2, req2, nil, []string{"de_DE", "fr_FR"})
	if !strings.Contains(b2.String(), `value="de_DE" selected="selected"`) {
		t.Fatalf("expected sanitized query selected: %q", b2.String())
	}
}
