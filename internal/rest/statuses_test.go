package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestStatusesListAnon(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/statuses", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]rest.PostStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	pub, ok := out["publish"]
	if !ok {
		t.Fatalf("missing publish; have %v", out)
	}
	if pub.Slug != "publish" || pub.Name != "Published" || !pub.Public || !pub.Queryable {
		t.Fatalf("publish=%+v", pub)
	}
	if pub.Private != nil || pub.Protected != nil || pub.ShowInList != nil {
		t.Fatalf("view context should omit edit fields: %+v", pub)
	}
	if _, ok := out["draft"]; ok {
		t.Fatal("anon should not see draft")
	}
}

func TestStatusesItemPublish(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/statuses/publish", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var st rest.PostStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatalf("json: %v", err)
	}
	if st.Slug != "publish" || st.Name != "Published" {
		t.Fatalf("%+v", st)
	}
}

func TestStatusesEditContextRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/statuses?context=edit", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStatusesOptionsAndHead(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodOptions, "/wp-json/wp/v2/statuses", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("OPTIONS status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != "GET" {
		t.Fatalf("Allow=%q", rec.Header().Get("Allow"))
	}
	var schema map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &schema); err != nil {
		t.Fatalf("json: %v", err)
	}
	if schema["title"] != "status" {
		t.Fatalf("schema title=%v", schema["title"])
	}

	req = httptest.NewRequest(http.MethodHead, "/wp-json/wp/v2/statuses", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD status=%d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body should be empty, got %q", rec.Body.String())
	}
}

func TestStatusesItemOptionsAndInvalidContext(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodOptions, "/wp-json/wp/v2/statuses/publish", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK || rec.Header().Get("Allow") != "GET" {
		t.Fatalf("OPTIONS item status=%d allow=%q body=%s", rec.Code, rec.Header().Get("Allow"), rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/statuses?context=bogus", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid context status=%d body=%s", rec.Code, rec.Body.String())
	}
}
