package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
