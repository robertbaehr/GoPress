package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestTemplatesRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/templates", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTemplatesAndPartsList(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/templates", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("templates status=%d body=%s", rec.Code, rec.Body.String())
	}
	var templates []rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &templates); err != nil {
		t.Fatal(err)
	}
	if len(templates) < 2 {
		t.Fatalf("expected theme templates, got %d", len(templates))
	}
	foundIndex := false
	for _, tpl := range templates {
		if tpl.Type != "wp_template" {
			t.Fatalf("type=%q", tpl.Type)
		}
		if tpl.Slug == "index" {
			foundIndex = true
			if tpl.WPID != 443 && !strings.Contains(tpl.ID, "index") {
				t.Fatalf("index template unexpected: %#v", tpl)
			}
		}
	}
	if !foundIndex {
		t.Fatalf("missing index template in %#v", templates)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/template-parts", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("parts status=%d body=%s", rec.Code, rec.Body.String())
	}
	var parts []rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &parts); err != nil {
		t.Fatal(err)
	}
	if len(parts) < 1 {
		t.Fatal("expected template parts")
	}
	foundHeader := false
	for _, p := range parts {
		if p.Type != "wp_template_part" {
			t.Fatalf("type=%q", p.Type)
		}
		if p.Slug == "header" {
			foundHeader = true
			if p.Area != "header" {
				t.Fatalf("header area=%q", p.Area)
			}
		}
	}
	if !foundHeader {
		t.Fatalf("missing header part in %#v", parts)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/templates/twentytwentyfive//index", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("item status=%d body=%s", rec.Code, rec.Body.String())
	}
}
