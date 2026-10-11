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

// P7.154: anonymous custom-header.php redirects to login.
func TestCustomHeaderRedirectsAnonymousP7154(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/custom-header.php", nil)
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

// P7.154: admin can view Header screen; save text color with nonce.
func TestCustomHeaderSaveTextColorP7154(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	getReq := httptest.NewRequest(http.MethodGet, "/wp-admin/custom-header.php", nil)
	getReq.AddCookie(ck)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%q", getRec.Code, getRec.Body.String())
	}
	body := getRec.Body.String()
	for _, want := range []string{
		`id="custom-header"`,
		`Header`,
		`name="header-text-color"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
	if getRec.Header().Get("X-GoPress-Custom-Header") != "ok" {
		t.Fatalf("header=%q", getRec.Header().Get("X-GoPress-Custom-Header"))
	}

	nonce := extractInputValue(body, "_wpnonce")
	if nonce == "" {
		t.Fatal("missing _wpnonce in Header form")
	}
	form := url.Values{}
	form.Set("action", "save")
	form.Set("_wpnonce", nonce)
	form.Set("header-text-color", "a1b2c3")
	postReq := httptest.NewRequest(http.MethodPost, "/wp-admin/custom-header.php", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(ck)
	postRec := httptest.NewRecorder()
	srv.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status=%d body=%q", postRec.Code, postRec.Body.String())
	}
	if got := wp.GetHeaderTextColor(c); got != "a1b2c3" {
		t.Fatalf("GetHeaderTextColor=%q", got)
	}
	if postRec.Header().Get("X-GoPress-Header-Text-Color") != "a1b2c3" {
		t.Fatalf("color header=%q", postRec.Header().Get("X-GoPress-Header-Text-Color"))
	}
	if !strings.Contains(postRec.Body.String(), "Header updated") {
		t.Fatalf("missing success notice: %q", postRec.Body.String())
	}

	t.Cleanup(func() {
		_ = wp.SetHeaderTextColor(c, "")
		_ = wp.SetHeaderImage(c, "")
	})
}

// P7.154: HasCustomHeader requires theme support.
func TestHasCustomHeaderNeedsSupportP7154(t *testing.T) {
	c := bootOrSkip(t)
	_ = wp.SetHeaderTextColor(c, "ff0000")
	t.Cleanup(func() { _ = wp.SetHeaderTextColor(c, "") })
	if wp.HasCustomHeader(c) {
		t.Fatal("without theme support, HasCustomHeader should be false")
	}
	wp.AddThemeSupport(c, "custom-header", map[string]any{"default-text-color": "ffffff"})
	if !wp.HasCustomHeader(c) {
		t.Fatal("expected HasCustomHeader with support + color")
	}
	classes := wp.GetBodyClass(c, "home")
	if !strings.Contains(classes, "custom-header") {
		t.Fatalf("body class missing custom-header: %q", classes)
	}
}
