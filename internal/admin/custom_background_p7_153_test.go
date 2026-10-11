package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.153: anonymous custom-background.php redirects to login.
func TestCustomBackgroundRedirectsAnonymousP7153(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/custom-background.php", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "wp-login.php") {
		t.Fatalf("Location=%q", loc)
	}
}

// P7.153: admin can view Background screen; save color with nonce.
func TestCustomBackgroundSaveColorP7153(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	getReq := httptest.NewRequest(http.MethodGet, "/wp-admin/custom-background.php", nil)
	getReq.AddCookie(ck)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%q", getRec.Code, getRec.Body.String())
	}
	body := getRec.Body.String()
	for _, want := range []string{
		`id="custom-background"`,
		`Background`,
		`name="background-color"`,
		`X-GoPress-Custom-Background`,
	} {
		if want == `X-GoPress-Custom-Background` {
			if getRec.Header().Get("X-GoPress-Custom-Background") != "ok" {
				t.Fatalf("header=%q", getRec.Header().Get("X-GoPress-Custom-Background"))
			}
			continue
		}
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}

	nonce := extractInputValue(body, "_wpnonce")
	if nonce == "" {
		t.Fatal("missing _wpnonce in Background form")
	}
	form := url.Values{}
	form.Set("action", "save")
	form.Set("_wpnonce", nonce)
	form.Set("background-color", "1a2b3c")
	postReq := httptest.NewRequest(http.MethodPost, "/wp-admin/custom-background.php", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(ck)
	postRec := httptest.NewRecorder()
	srv.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status=%d body=%q", postRec.Code, postRec.Body.String())
	}
	if got := wp.GetBackgroundColor(c); got != "1a2b3c" {
		t.Fatalf("GetBackgroundColor=%q", got)
	}
	if postRec.Header().Get("X-GoPress-Background-Color") != "1a2b3c" {
		t.Fatalf("color header=%q", postRec.Header().Get("X-GoPress-Background-Color"))
	}
	if !strings.Contains(postRec.Body.String(), "Background updated") {
		t.Fatalf("missing success notice: %q", postRec.Body.String())
	}

	// Cleanup theme mod so later tests / corpus stay clean.
	t.Cleanup(func() {
		_ = wp.SetBackgroundColor(c, "")
		_ = wp.SetBackgroundImage(c, "")
	})
}

// P7.153: HasCustomBackground requires theme support.
func TestHasCustomBackgroundNeedsSupportP7153(t *testing.T) {
	c := bootOrSkip(t)
	_ = wp.SetBackgroundColor(c, "ff0000")
	t.Cleanup(func() { _ = wp.SetBackgroundColor(c, "") })
	if wp.HasCustomBackground(c) {
		t.Fatal("without theme support, HasCustomBackground should be false")
	}
	wp.AddThemeSupport(c, "custom-background", map[string]any{"default-color": "ffffff"})
	if !wp.HasCustomBackground(c) {
		t.Fatal("expected HasCustomBackground with support + color")
	}
	classes := wp.GetBodyClass(c, "home")
	if !strings.Contains(classes, "custom-background") {
		t.Fatalf("body class missing custom-background: %q", classes)
	}
}
