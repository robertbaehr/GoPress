package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestGlobalStylesRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/global-styles/themes/twentytwentyfive", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGlobalStylesThemeAndItem(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/global-styles/themes/twentytwentyfive", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("theme status=%d body=%s", rec.Code, rec.Body.String())
	}
	var theme rest.GlobalStylesTheme
	if err := json.Unmarshal(rec.Body.Bytes(), &theme); err != nil {
		t.Fatal(err)
	}
	if theme.Settings == nil || theme.Styles == nil {
		t.Fatalf("expected settings/styles, got %#v", theme)
	}
	if theme.Links == nil || theme.Links["self"] == nil {
		t.Fatalf("expected _links.self: %#v", theme.Links)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/global-styles/445", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("item status=%d body=%s", rec.Code, rec.Body.String())
	}
	var item rest.GlobalStylesItem
	if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID != 445 {
		t.Fatalf("id=%d", item.ID)
	}
	if item.Title["raw"] == "" {
		t.Fatalf("missing title: %#v", item.Title)
	}
	if item.Styles == nil || item.Settings == nil {
		t.Fatalf("expected settings/styles: %#v", item)
	}
	bg, _ := item.Styles["color"].(map[string]any)
	if bg == nil || bg["background"] != "#f7f7f7" {
		t.Fatalf("expected fixture background, got %#v", item.Styles)
	}
}

func TestGlobalStylesThemeVariations(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/global-styles/themes/twentytwentyfive/variations", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var vars []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &vars); err != nil {
		t.Fatal(err)
	}
	if len(vars) == 0 {
		t.Fatal("expected theme style variations for twentytwentyfive")
	}
	found := false
	for _, v := range vars {
		if title, _ := v["title"].(string); title == "Evening" {
			found = true
			if v["settings"] == nil {
				t.Fatalf("Evening variation missing settings: %#v", v)
			}
		}
	}
	if !found {
		t.Fatalf("expected Evening variation, got %#v", vars)
	}
}
