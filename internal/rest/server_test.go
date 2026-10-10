package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIndexAnonShape(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var idx rest.Index
	if err := json.Unmarshal(rec.Body.Bytes(), &idx); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if idx.Name == "" {
		t.Fatal("expected name")
	}
	if !containsAll(idx.Namespaces, []string{"wp/v2"}) {
		t.Fatalf("namespaces=%v", idx.Namespaces)
	}
	wantRoutes := []string{
		"/", "/wp/v2",
		"/oembed/1.0", "/oembed/1.0/embed",
		"/wp/v2/posts", "/wp/v2/posts/(?P<id>[\\d]+)",
		"/wp/v2/pages", "/wp/v2/pages/(?P<id>[\\d]+)",
		"/wp/v2/types",
		"/wp/v2/taxonomies",
		"/wp/v2/categories", "/wp/v2/categories/(?P<id>[\\d]+)",
		"/wp/v2/tags", "/wp/v2/tags/(?P<id>[\\d]+)",
		"/wp/v2/comments", "/wp/v2/comments/(?P<id>[\\d]+)",
		"/wp/v2/media", "/wp/v2/media/(?P<id>[\\d]+)",
		"/wp/v2/search",
		"/wp/v2/blocks", "/wp/v2/blocks/(?P<id>[\\d]+)",
		"/wp/v2/block-types",
		"/wp/v2/block-renderer/(?P<name>[a-z0-9-]+/[a-z0-9-]+)",
		"/wp/v2/settings",
		"/wp/v2/users/me",
	}
	for _, r := range wantRoutes {
		ri, ok := idx.Routes[r]
		if !ok {
			t.Fatalf("missing route %q; have %v", r, routeKeys(idx.Routes))
		}
		if len(ri.Methods) == 0 {
			t.Fatalf("route %q has no methods", r)
		}
	}

	// Route keys must be a subset of the golden anon index.
	golden := loadGoldenRouteKeys(t)
	for k := range idx.Routes {
		if _, ok := golden[k]; !ok {
			t.Fatalf("route %q not in golden index-anon.json", k)
		}
	}
	// Namespaces must be a subset of golden namespaces.
	goldenNS := loadGoldenNamespaces(t)
	for _, ns := range idx.Namespaces {
		if !goldenNS[ns] {
			t.Fatalf("namespace %q not in golden", ns)
		}
	}
}

func TestIndexWPJSONNoTrailingSlash(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestIndexAdminAddsRoutes(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin user unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var idx rest.Index
	if err := json.Unmarshal(rec.Body.Bytes(), &idx); err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{"/wp/v2/settings", "/wp/v2/users/me", "/wp/v2/plugins"} {
		if _, ok := idx.Routes[r]; !ok {
			t.Fatalf("logged-in index missing %q", r)
		}
	}
	golden := loadGoldenRouteKeys(t)
	for k := range idx.Routes {
		if _, ok := golden[k]; !ok {
			t.Fatalf("admin route %q not in golden", k)
		}
	}
}

func bootOrSkip(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH:        "/workspace/",
		AuthKey:        "gopress-ref-auth-key-0001-fixed-for-oracle",
		SecureAuthKey:  "gopress-ref-secure-auth-key-0002-fixed",
		LoggedInKey:    "gopress-ref-logged-in-key-0003-fixed",
		NonceKey:       "gopress-ref-nonce-key-0004-fixed-oracle",
		AuthSalt:       "gopress-ref-auth-salt-0005-fixed-oracle",
		SecureAuthSalt: "gopress-ref-secure-auth-salt-0006-fixed",
		LoggedInSalt:   "gopress-ref-logged-in-salt-0007-fixed",
		NonceSalt:      "gopress-ref-nonce-salt-0008-fixed-oracle",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	return c
}

func loadGoldenRouteKeys(t *testing.T) map[string]bool {
	t.Helper()
	path := filepath.Join("..", "..", "tests", "golden", "rest", "index-anon.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var doc struct {
		Routes map[string]json.RawMessage `json:"routes"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	out := make(map[string]bool, len(doc.Routes))
	for k := range doc.Routes {
		out[k] = true
	}
	return out
}

func loadGoldenNamespaces(t *testing.T) map[string]bool {
	t.Helper()
	path := filepath.Join("..", "..", "tests", "golden", "rest", "index-anon.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var doc struct {
		Namespaces []string `json:"namespaces"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	out := make(map[string]bool, len(doc.Namespaces))
	for _, ns := range doc.Namespaces {
		out[ns] = true
	}
	return out
}

func containsAll(have, want []string) bool {
	set := make(map[string]bool, len(have))
	for _, h := range have {
		set[h] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

func routeKeys(m map[string]rest.RouteIndex) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
