package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestFrontAdminBarWhenLoggedIn(t *testing.T) {
	c := testBoot(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="wpadminbar"`) {
		t.Fatalf("expected admin bar, got %q", body)
	}
	if !strings.Contains(body, "admin-bar") {
		t.Fatalf("expected admin-bar body class, got %q", body)
	}
	if !strings.Contains(body, "Howdy, admin") {
		t.Fatalf("expected howdy, got %q", body)
	}
}

func TestFrontAdminBarAbsentWhenAnonymous(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, `id="wpadminbar"`) {
		t.Fatalf("anonymous should not see admin bar: %q", body)
	}
}
