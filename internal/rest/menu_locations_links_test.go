package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestMenuLocationAnonMessage(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/menu-locations", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	msg, _ := body["message"].(string)
	if !strings.Contains(msg, "menu locations") {
		t.Fatalf("expected menu locations message, got %q", msg)
	}
}

func TestMenuLocationLinks(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() {
		_ = wp.SetNavMenuLocation(c, "primary", 0)
		_ = c.DB.Close()
	})
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}
	_ = wp.SetNavMenuLocation(c, "primary", 37)

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/menu-locations/primary", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var loc rest.MenuLocation
	if err := json.Unmarshal(rec.Body.Bytes(), &loc); err != nil {
		t.Fatal(err)
	}
	if loc.Links == nil || loc.Links["self"] == nil || loc.Links["collection"] == nil {
		t.Fatalf("expected self/collection links: %+v", loc.Links)
	}
	if loc.Links["https://api.w.org/menu"] == nil {
		t.Fatalf("expected menu link when assigned: %+v", loc.Links)
	}
}

func TestMenuLocationReadAccessFilter(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	c.Hooks.AddFilter("rest_menu_read_access", func(value any, args ...any) any { return true }, 10, 1)
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/menu-locations", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("filter should grant read: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
