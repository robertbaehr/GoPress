package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestTemplatesDeleteForce(t *testing.T) {
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
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	slug := "gopress-p473-del"
	createBody := `{"slug":"` + slug + `","title":{"raw":"P473 Delete"},"content":{"raw":"<!-- wp:paragraph --><p>Bye</p><!-- /wp:paragraph -->"}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/templates", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if c.DB != nil && created.WPID > 0 {
			q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", created.WPID)
			if err == nil {
				_, _ = c.DB.Exec(q, args...)
			}
		}
	})

	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/templates/"+created.ID, nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("trash status=%d body=%s", rec.Code, rec.Body.String())
	}
	var trashed rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &trashed); err != nil {
		t.Fatal(err)
	}
	if trashed.Status != "trash" {
		t.Fatalf("expected trash status, got %#v", trashed)
	}

	// Recreate for force-delete path (trashed row still exists).
	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/templates/"+created.ID+"?force=true", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("force delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["deleted"] != true {
		t.Fatalf("expected deleted=true: %#v", resp)
	}
	created.WPID = 0 // already deleted
}

func TestTemplatesDeleteThemeRejected(t *testing.T) {
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
	// Theme-sourced home template (or first theme template from GET).
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/templates", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d", rec.Code)
	}
	var list []rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	var themeID string
	for _, tplx := range list {
		if tplx.Source == "theme" {
			themeID = tplx.ID
			break
		}
	}
	if themeID == "" {
		t.Skip("no theme templates in fixture")
	}
	req = httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/templates/"+themeID+"?force=true", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for theme template, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTemplatesDeleteRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/templates/twentytwentyfive//home?force=true", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
