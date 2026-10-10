package login

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestWriteLanguageSwitcherEmpty(t *testing.T) {
	var b strings.Builder
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	writeLanguageSwitcher(&b, req, nil, nil)
	if b.Len() != 0 {
		t.Fatalf("expected no markup without languages, got %q", b.String())
	}
}

func TestWriteLanguageSwitcherWithLocales(t *testing.T) {
	var b strings.Builder
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?redirect_to=%2Fwp-admin%2F", nil)
	writeLanguageSwitcher(&b, req, nil, []string{"de_DE", "fr_FR"})
	out := b.String()
	if !strings.Contains(out, `class="language-switcher"`) {
		t.Fatalf("missing language-switcher: %q", out)
	}
	if !strings.Contains(out, `name="wp_lang"`) || !strings.Contains(out, `id="language-switcher-locales"`) {
		t.Fatalf("missing select: %q", out)
	}
	if !strings.Contains(out, "English (United States)") {
		t.Fatalf("missing en_US option: %q", out)
	}
	if !strings.Contains(out, `value="de_DE"`) || !strings.Contains(out, `value="fr_FR"`) {
		t.Fatalf("missing locale options: %q", out)
	}
	if !strings.Contains(out, `name="redirect_to"`) {
		t.Fatalf("missing redirect_to hidden: %q", out)
	}
	if !strings.Contains(out, `value="Change"`) {
		t.Fatalf("missing Change button: %q", out)
	}
}

func TestWriteLanguageSwitcherPrefersCookie(t *testing.T) {
	var b strings.Builder
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	req.AddCookie(&http.Cookie{Name: "wp_lang", Value: "de_DE"})
	writeLanguageSwitcher(&b, req, nil, []string{"de_DE", "fr_FR"})
	out := b.String()
	if !strings.Contains(out, `value="de_DE" selected="selected"`) {
		t.Fatalf("expected de_DE selected from cookie: %q", out)
	}
}

func TestWriteLanguageSwitcherPreservesPOSTFields(t *testing.T) {
	var b strings.Builder
	form := url.Values{}
	form.Set("redirect_to", "/wp-admin/")
	form.Set("action", "login")
	form.Set("interim-login", "1")
	form.Set("wp_lang", "fr_FR")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	writeLanguageSwitcher(&b, req, nil, []string{"de_DE", "fr_FR"})
	out := b.String()
	if !strings.Contains(out, `value="fr_FR" selected="selected"`) {
		t.Fatalf("expected fr_FR from POST: %q", out)
	}
	if !strings.Contains(out, `name="redirect_to"`) || !strings.Contains(out, `/wp-admin/`) {
		t.Fatalf("expected redirect_to from POST: %q", out)
	}
	if !strings.Contains(out, `name="action"`) || !strings.Contains(out, `value="login"`) {
		t.Fatalf("expected action from POST: %q", out)
	}
	if !strings.Contains(out, `name="interim-login"`) || !strings.Contains(out, `value="1"`) {
		t.Fatalf("expected interim-login from POST: %q", out)
	}
}

func TestWriteLanguageSwitcherDisplayFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("login_display_language_dropdown", func(value any, args ...any) any {
		return false
	}, 10, 1)
	var b strings.Builder
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	writeLanguageSwitcher(&b, req, c, []string{"de_DE"})
	if b.Len() != 0 {
		t.Fatalf("filter false should hide switcher: %q", b.String())
	}
}

func TestWriteLanguageSwitcherPreservesReauthAndArgsFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("login_language_dropdown_args", func(value any, args ...any) any {
		return map[string]any{
			"id":       "custom-lang",
			"name":     "wp_lang",
			"selected": "fr_FR",
		}
	}, 10, 1)
	var b strings.Builder
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?reauth=1&redirect_to=%2Fwp-admin%2F", nil)
	writeLanguageSwitcher(&b, req, c, []string{"de_DE", "fr_FR"})
	out := b.String()
	if !strings.Contains(out, `id="custom-lang"`) || !strings.Contains(out, `for="custom-lang"`) {
		t.Fatalf("expected filtered id: %q", out)
	}
	if !strings.Contains(out, `value="fr_FR" selected="selected"`) {
		t.Fatalf("expected filtered selected: %q", out)
	}
	if !strings.Contains(out, `name="reauth"`) || !strings.Contains(out, `value="1"`) {
		t.Fatalf("expected reauth hidden: %q", out)
	}
}

func TestWriteLoginHTMLSetsLangCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?wp_lang=fr_FR", nil)
	rec := httptest.NewRecorder()
	writeLoginHTML(rec, req, "<html></html>", false, nil)
	found := false
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "wp_lang" {
			found = true
			if ck.Value != "fr_FR" {
				t.Fatalf("wp_lang=%q", ck.Value)
			}
			if !ck.HttpOnly {
				t.Fatal("expected HttpOnly wp_lang cookie")
			}
		}
	}
	if !found {
		t.Fatal("missing wp_lang Set-Cookie")
	}
	// Sanitized value
	req = httptest.NewRequest(http.MethodGet, "/wp-login.php?wp_lang="+wp.SanitizeTextField("en_US"), nil)
	rec = httptest.NewRecorder()
	writeLoginHTML(rec, req, "ok", false, nil)
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "wp_lang" && ck.Value == "" {
			t.Fatal("empty wp_lang cookie")
		}
	}
}
