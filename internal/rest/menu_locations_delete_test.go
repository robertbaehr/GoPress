package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestMenuLocationDeleteUnsets(t *testing.T) {
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

	body := `{"menu":37}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/menu-locations/primary", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("assign status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/menu-locations/primary", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var loc rest.MenuLocation
	if err := json.Unmarshal(rec.Body.Bytes(), &loc); err != nil {
		t.Fatal(err)
	}
	if loc.Name != "primary" || loc.Menu != 0 {
		t.Fatalf("expected unset location, got %+v", loc)
	}
	if got := wp.GetNavMenuLocations(c)["primary"]; got != 0 {
		t.Fatalf("theme_mod still set: %d", got)
	}
	allow := rec.Header().Get("Allow")
	if !strings.Contains(allow, "GET") || !strings.Contains(allow, "DELETE") {
		t.Fatalf("Allow=%q", allow)
	}
}

func TestMenuLocationDeleteRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/menu-locations/primary", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
