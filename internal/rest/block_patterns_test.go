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

func TestBlockPatternCategoriesRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-patterns/categories", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var errBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody["code"] != "rest_cannot_view" {
		t.Fatalf("code=%v", errBody["code"])
	}
}

func TestBlockPatternCategoriesList(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-patterns/categories", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cats []wp.PatternCategory
	if err := json.Unmarshal(rec.Body.Bytes(), &cats); err != nil {
		t.Fatal(err)
	}
	if len(cats) < 20 {
		t.Fatalf("expected core categories, got %d", len(cats))
	}
	found := map[string]bool{}
	for _, cat := range cats {
		if cat.Name == "" || cat.Label == "" {
			t.Fatalf("incomplete category %#v", cat)
		}
		found[cat.Name] = true
	}
	for _, want := range []string{"banner", "featured", "posts", "twentytwentyfive_page"} {
		if !found[want] {
			t.Fatalf("missing category %q in %#v", want, found)
		}
	}
}

func TestBlockPatternsList(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-patterns/patterns", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var patterns []rest.RESTBlockPattern
	if err := json.Unmarshal(rec.Body.Bytes(), &patterns); err != nil {
		t.Fatal(err)
	}
	if len(patterns) < 5 {
		t.Fatalf("expected registered patterns, got %d", len(patterns))
	}
	for i := 1; i < len(patterns); i++ {
		if patterns[i-1].Name >= patterns[i].Name {
			t.Fatalf("unsorted patterns: %q then %q", patterns[i-1].Name, patterns[i].Name)
		}
	}
}

func TestMigratePatternCategories(t *testing.T) {
	// Exercised via list; also check anonymous patterns endpoint.
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-patterns/patterns", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestBlockPatternsHEAD(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodHead, "/wp-json/wp/v2/block-patterns/patterns", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "HEAD") {
		t.Fatalf("Allow=%q want HEAD", allow)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD should have empty body, got %q", rec.Body.String())
	}
}

func TestBlockPatternCategoriesOPTIONSAndContext(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodOptions, "/wp-json/wp/v2/block-patterns/categories", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("OPTIONS status=%d body=%s", rec.Code, rec.Body.String())
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "OPTIONS") {
		t.Fatalf("Allow=%q", allow)
	}
	var schema map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}
	if schema["title"] != "block-pattern-category" {
		t.Fatalf("schema title=%v", schema["title"])
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-patterns/categories?context=bogus", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bogus context status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "rest_invalid_param") {
		t.Fatalf("body=%s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-patterns/categories?context=edit", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit context status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cats []rest.RESTBlockPatternCategory
	if err := json.Unmarshal(rec.Body.Bytes(), &cats); err != nil {
		t.Fatal(err)
	}
	if len(cats) < 5 {
		t.Fatalf("expected categories, got %d", len(cats))
	}
	// description key always present (may be empty).
	raw := rec.Body.String()
	if !strings.Contains(raw, `"description"`) {
		t.Fatalf("expected description field: %s", raw[:min(200, len(raw))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestBlockPatternSchemaFields(t *testing.T) {
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
	// Register a pattern with P4.106 schema fields.
	if c.Patterns == nil {
		c.Patterns = wp.NewPatternRegistry()
	}
	c.Patterns.RegisterBlockPattern("gopress/test-rich", wp.BlockPattern{
		Name:          "gopress/test-rich",
		Title:         "Rich",
		Content:       "<!-- wp:paragraph --><p>Hi</p><!-- /wp:paragraph -->",
		Inserter:      true,
		Keywords:      []string{"demo", "fixture"},
		PostTypes:     []string{"post"},
		TemplateTypes: []string{"single"},
		ViewportWidth: 1200,
		Source:        "theme",
	})
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-patterns/patterns", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var patterns []rest.RESTBlockPattern
	if err := json.Unmarshal(rec.Body.Bytes(), &patterns); err != nil {
		t.Fatal(err)
	}
	var found *rest.RESTBlockPattern
	for i := range patterns {
		if patterns[i].Name == "gopress/test-rich" {
			found = &patterns[i]
			break
		}
	}
	if found == nil {
		t.Fatal("missing gopress/test-rich pattern")
	}
	if found.ViewportWidth != 1200 {
		t.Fatalf("viewport_width=%d", found.ViewportWidth)
	}
	if len(found.Keywords) != 2 || found.Keywords[0] != "demo" {
		t.Fatalf("keywords=%v", found.Keywords)
	}
	if len(found.PostTypes) != 1 || found.PostTypes[0] != "post" {
		t.Fatalf("post_types=%v", found.PostTypes)
	}
	if len(found.TemplateTypes) != 1 || found.TemplateTypes[0] != "single" {
		t.Fatalf("template_types=%v", found.TemplateTypes)
	}
}
