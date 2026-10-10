package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func keysOf(m map[string]rest.PostType) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
