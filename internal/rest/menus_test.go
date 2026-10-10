package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestNavigationRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/navigation", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestNavigationListAndItem(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/navigation", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var menus []rest.NavigationMenu
	if err := json.Unmarshal(rec.Body.Bytes(), &menus); err != nil {
		t.Fatal(err)
	}
	if len(menus) < 1 {
		t.Fatalf("expected wp_navigation posts, got %d", len(menus))
	}
	found := false
	for _, m := range menus {
		if m.Type != "wp_navigation" {
			t.Fatalf("type=%q", m.Type)
		}
		if m.Slug == "fixture-navigation" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing fixture-navigation in %#v", menus)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/navigation/446", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("item status=%d body=%s", rec.Code, rec.Body.String())
	}
	var one rest.NavigationMenu
	if err := json.Unmarshal(rec.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if one.ID != 446 || one.Slug != "fixture-navigation" {
		t.Fatalf("got %#v", one)
	}
	if one.Content.Raw == nil || !strings.Contains(*one.Content.Raw, "navigation-link") {
		t.Fatalf("expected navigation-link content, got %#v", one.Content)
	}
}

func TestClassicMenusReadAPI(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/menus", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("menus status=%d body=%s", rec.Code, rec.Body.String())
	}
	var menus []rest.ClassicMenu
	if err := json.Unmarshal(rec.Body.Bytes(), &menus); err != nil {
		t.Fatal(err)
	}
	if len(menus) < 1 || menus[0].Slug != "fixture-classic-menu" {
		t.Fatalf("expected fixture-classic-menu, got %#v", menus)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/menus/37", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("menu item status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/menu-items", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("menu-items status=%d body=%s", rec.Code, rec.Body.String())
	}
	var items []rest.ClassicMenuItem
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) < 2 {
		t.Fatalf("expected menu items, got %#v", items)
	}
	var child *rest.ClassicMenuItem
	for i := range items {
		if items[i].ID == 448 {
			child = &items[i]
			break
		}
	}
	if child == nil || child.Parent != 447 || child.Menus != 37 {
		t.Fatalf("child item=%#v", child)
	}
	if child.URL == "" {
		t.Fatal("expected menu item URL")
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/menu-locations", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("locations status=%d body=%s", rec.Code, rec.Body.String())
	}
}
