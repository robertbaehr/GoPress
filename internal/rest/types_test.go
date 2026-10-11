package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestTypesList(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/types", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var types map[string]rest.PostType
	if err := json.Unmarshal(rec.Body.Bytes(), &types); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	for _, slug := range []string{"post", "page", "attachment"} {
		pt, ok := types[slug]
		if !ok {
			t.Fatalf("missing type %q; have %v", slug, keysOf(types))
		}
		if pt.Slug != slug {
			t.Fatalf("%s slug=%q", slug, pt.Slug)
		}
		if pt.RestBase == "" || pt.RestNamespace != "wp/v2" {
			t.Fatalf("%s rest_base=%q rest_namespace=%q", slug, pt.RestBase, pt.RestNamespace)
		}
		if pt.Name == "" {
			t.Fatalf("%s empty name", slug)
		}
		if pt.Viewable != nil || pt.Supports != nil || pt.Labels != nil {
			t.Fatalf("%s view context should omit edit fields", slug)
		}
	}
	if !types["page"].Hierarchical {
		t.Fatal("page should be hierarchical")
	}
	if types["post"].Hierarchical {
		t.Fatal("post should not be hierarchical")
	}
	if types["attachment"].RestBase != "media" {
		t.Fatalf("attachment rest_base=%q", types["attachment"].RestBase)
	}
}

func TestTypesItemAndOptions(t *testing.T) {
	// P4.125: single type, invalid, OPTIONS schema, HEAD, edit context.
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/types/post", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("item status=%d body=%s", rec.Code, rec.Body.String())
	}
	var pt rest.PostType
	if err := json.Unmarshal(rec.Body.Bytes(), &pt); err != nil {
		t.Fatal(err)
	}
	if pt.Slug != "post" || pt.RestBase != "posts" || pt.Name != "Posts" {
		t.Fatalf("post item: %#v", pt)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/types/not-a-type", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "rest_type_invalid") {
		t.Fatalf("invalid type: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodOptions, "/wp-json/wp/v2/types", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"title":"type"`) {
		t.Fatalf("OPTIONS: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/types/post?context=edit", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "rest_forbidden_context") {
		t.Fatalf("anon edit: %d %s", rec.Code, rec.Body.String())
	}

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req = httptest.NewRequest(http.MethodHead, "/wp-json/wp/v2/types/page", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD: %d body=%q", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/types/post?context=edit", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin edit: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &pt); err != nil {
		t.Fatal(err)
	}
	if pt.Viewable == nil || !*pt.Viewable {
		t.Fatalf("expected viewable: %#v", pt)
	}
	if pt.Supports == nil || pt.Supports["title"] != true {
		t.Fatalf("expected supports: %#v", pt.Supports)
	}
	if pt.Visibility == nil || !pt.Visibility.ShowUI {
		t.Fatalf("expected visibility: %#v", pt.Visibility)
	}
	if pt.Labels == nil || pt.Capabilities == nil {
		t.Fatalf("expected labels/capabilities: %#v", pt)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/types?context=bogus", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bogus context: %d", rec.Code)
	}
}

func keysOf(m map[string]rest.PostType) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
