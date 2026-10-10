package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestTemplatesCreateRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/templates",
		strings.NewReader(`{"slug":"gopress-test-tpl","title":"GoPress Test","content":"<!-- wp:paragraph --><p>Hi</p><!-- /wp:paragraph -->"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTemplatesCreateAndUpdate(t *testing.T) {
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

	slug := "gopress-p452-tpl"
	createBody := `{"slug":"` + slug + `","title":{"raw":"P452 Template"},"content":{"raw":"<!-- wp:paragraph --><p>One</p><!-- /wp:paragraph -->"}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/templates", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Slug != slug {
		t.Fatalf("slug=%q", created.Slug)
	}
	if created.WPID <= 0 {
		t.Fatalf("expected wp_id, got %#v", created)
	}
	if created.Source != "custom" {
		t.Fatalf("source=%q", created.Source)
	}
	t.Cleanup(func() {
		if c.DB != nil && created.WPID > 0 {
			q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", created.WPID)
			if err == nil {
				_, _ = c.DB.Exec(q, args...)
			}
		}
	})

	updateBody := `{"content":{"raw":"<!-- wp:paragraph --><p>Two</p><!-- /wp:paragraph -->"},"title":"P452 Updated"}`
	req = httptest.NewRequest(http.MethodPut, "/wp-json/wp/v2/templates/"+created.ID, strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Title["raw"] != "P452 Updated" {
		t.Fatalf("title=%#v", updated.Title)
	}
	raw, _ := updated.Content["raw"].(string)
	if !strings.Contains(raw, "<p>Two</p>") {
		t.Fatalf("content=%q", raw)
	}
}

func TestTemplatesUpdateThemeFileCreatesCustom(t *testing.T) {
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
		t.Fatalf("list status=%d", rec.Code)
	}
	var templates []rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &templates); err != nil {
		t.Fatal(err)
	}
	var themeTpl *rest.BlockTemplate
	for i := range templates {
		if templates[i].Source == "theme" && templates[i].Slug == "404" {
			themeTpl = &templates[i]
			break
		}
	}
	if themeTpl == nil {
		t.Skip("no theme 404 template in fixture")
	}

	updateBody := `{"content":{"raw":"<!-- wp:paragraph --><p>Custom 404</p><!-- /wp:paragraph -->"}}`
	req = httptest.NewRequest(http.MethodPatch, "/wp-json/wp/v2/templates/"+themeTpl.ID, strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated rest.BlockTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Source != "custom" {
		t.Fatalf("expected custom source after override, got %q", updated.Source)
	}
	if updated.WPID <= 0 {
		t.Fatalf("expected wp_id after override: %#v", updated)
	}
	t.Cleanup(func() {
		if c.DB != nil && updated.WPID > 0 {
			q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", updated.WPID)
			if err == nil {
				_, _ = c.DB.Exec(q, args...)
			}
		}
	})
}
