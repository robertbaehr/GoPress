package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestThemesActivateViaREST(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	prev := wp.ActiveStylesheet(c)
	if prev == "" {
		prev = "twentytwentyfive"
	}
	defer func() {
		_ = wp.SwitchTheme(c, "/workspace/", prev)
	}()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	srv := &rest.Server{}

	target := "twentytwentyfour"
	if target == prev {
		target = "twentytwentythree"
	}

	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/themes/"+target, bytes.NewBufferString(`{"status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c2 := c.ForRequest(rec, req)
	srv.Handle(rec, req, c2)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var th rest.ThemeREST
	if err := json.Unmarshal(rec.Body.Bytes(), &th); err != nil {
		t.Fatalf("json: %v", err)
	}
	if th.Stylesheet != target || th.Status != "active" {
		t.Fatalf("theme=%+v", th)
	}
	if wp.ActiveStylesheet(c) != target {
		t.Fatalf("option stylesheet=%q want %q", wp.ActiveStylesheet(c), target)
	}

	// Idempotent re-activate.
	req2 := httptest.NewRequest(http.MethodPut, "/wp-json/wp/v2/themes/"+target, bytes.NewBufferString(`{"status":"active"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec2 := httptest.NewRecorder()
	srv.Handle(rec2, req2, c.ForRequest(rec2, req2))
	if rec2.Code != http.StatusOK {
		t.Fatalf("re-activate status=%d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestThemesActivateRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/themes/twentytwentyfour", bytes.NewBufferString(`{"status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestThemesActivateRejectsInactive(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/themes/twentytwentyfour", bytes.NewBufferString(`{"status":"inactive"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	(&rest.Server{}).Handle(rec, req, c.ForRequest(rec, req))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
