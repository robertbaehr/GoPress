package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestBlockTypesRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-types", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBlockTypesList(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-types", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var types []rest.BlockType
	if err := json.Unmarshal(rec.Body.Bytes(), &types); err != nil {
		t.Fatal(err)
	}
	if len(types) < 5 {
		t.Fatalf("expected several block types, got %d", len(types))
	}
	found := false
	for _, bt := range types {
		if bt.Name == "core/paragraph" && bt.Title != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing core/paragraph in %#v", types)
	}
}

func TestBlockTypeItem(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-types/core/paragraph", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var bt rest.BlockType
	if err := json.Unmarshal(rec.Body.Bytes(), &bt); err != nil {
		t.Fatal(err)
	}
	if bt.Name != "core/paragraph" || bt.Title == "" {
		t.Fatalf("unexpected item %#v", bt)
	}
}

// TestBlockTypesLibraryRegister covers P5.70: more core block metadata
// (title/category/description) for blocks GoPress renders.
func TestBlockTypesLibraryRegister(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-types", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var types []rest.BlockType
	if err := json.Unmarshal(rec.Body.Bytes(), &types); err != nil {
		t.Fatal(err)
	}
	byName := map[string]rest.BlockType{}
	for _, bt := range types {
		byName[bt.Name] = bt
	}
	want := []struct {
		name, title, category string
		dynamic               bool
	}{
		{"core/table", "Table", "text", false},
		{"core/gallery", "Gallery", "media", false},
		{"core/query", "Query Loop", "theme", true},
		{"core/latest-posts", "Latest Posts", "widgets", true},
		{"core/site-logo", "Site Logo", "theme", true},
		{"core/social-links", "Social Icons", "widgets", false},
	}
	for _, w := range want {
		bt, ok := byName[w.name]
		if !ok {
			t.Fatalf("missing %s in block-types (%d total)", w.name, len(types))
		}
		if bt.Title != w.title || bt.Category != w.category {
			t.Fatalf("%s metadata: title=%q category=%q want %q/%q", w.name, bt.Title, bt.Category, w.title, w.category)
		}
		if bt.Description == "" {
			t.Fatalf("%s: expected non-empty description", w.name)
		}
		if bt.IsDynamic != w.dynamic {
			t.Fatalf("%s is_dynamic=%v want %v", w.name, bt.IsDynamic, w.dynamic)
		}
	}
	if len(types) < 40 {
		t.Fatalf("expected expanded library (≥40), got %d", len(types))
	}
}

func TestBlockTypesNamespacePath(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-types/core", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var types []rest.BlockType
	if err := json.Unmarshal(rec.Body.Bytes(), &types); err != nil {
		t.Fatal(err)
	}
	if len(types) < 3 {
		t.Fatalf("expected core namespace types, got %d", len(types))
	}
	for _, bt := range types {
		if !strings.HasPrefix(bt.Name, "core/") {
			t.Fatalf("non-core type in namespace list: %s", bt.Name)
		}
	}
}

func TestBlockTypeItemInvalid(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/block-types/core/not-a-real-block", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var errBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody["code"] != "rest_block_type_invalid" {
		t.Fatalf("code=%v", errBody["code"])
	}
}

func TestBlockRendererStub(t *testing.T) {
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
	body := `{"attributes":{"height":40}}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/block-renderer/core/spacer", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	rendered, _ := out["rendered"].(string)
	if rendered == "" {
		t.Fatalf("expected rendered HTML, got %v", out)
	}
	if !strings.Contains(rendered, "height:40px") {
		t.Fatalf("expected spacer height from POST attrs, got %q", rendered)
	}
}

func TestBlockRendererPostFormAndPostID(t *testing.T) {
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
	form := "attributes=" + url.QueryEscape(`{"height":"72px"}`) + "&post_id=1"
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/block-renderer/core/spacer", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	rendered, _ := out["rendered"].(string)
	if !strings.Contains(rendered, "height:72px") {
		t.Fatalf("expected form attributes rendered, got %q", rendered)
	}
}

func TestBlockRendererRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/block-renderer/core/spacer", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
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
	if errBody["code"] != "block_cannot_read" {
		t.Fatalf("code=%v want block_cannot_read", errBody["code"])
	}
}
