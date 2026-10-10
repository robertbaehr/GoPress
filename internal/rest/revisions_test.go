package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestRevisionsRequireAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13/revisions", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRevisionsOptionsContextPagination(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	// OPTIONS is public schema (no auth).
	req := httptest.NewRequest(http.MethodOptions, "/wp-json/wp/v2/posts/13/revisions", nil)
	rec := httptest.NewRecorder()
	(&rest.Server{}).Handle(rec, req, c.ForRequest(rec, req))
	if rec.Code != http.StatusOK {
		t.Fatalf("OPTIONS status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"title":"post-revision"`) {
		t.Fatalf("OPTIONS schema: %s", rec.Body.String())
	}
	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Fatalf("Allow=%q", got)
	}

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13/revisions?context=bogus", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	(&rest.Server{}).Handle(rec, req, c.ForRequest(rec, req))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "rest_invalid_param") {
		t.Fatalf("bad context status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodHead, "/wp-json/wp/v2/posts/13/revisions", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	(&rest.Server{}).Handle(rec, req, c.ForRequest(rec, req))
	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD status=%d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body should be empty: %q", rec.Body.String())
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("HEAD missing X-WP-Total")
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13/revisions?per_page=1", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	(&rest.Server{}).Handle(rec, req, c.ForRequest(rec, req))
	if rec.Code != http.StatusOK {
		t.Fatalf("per_page status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page []rest.Revision
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 {
		t.Fatalf("per_page=1 got %d", len(page))
	}
	if rec.Header().Get("X-WP-TotalPages") == "" || rec.Header().Get("X-WP-TotalPages") == "0" {
		t.Fatalf("TotalPages=%q", rec.Header().Get("X-WP-TotalPages"))
	}
}

func TestRevisionsListAndItem(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13/revisions", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var revs []rest.Revision
	if err := json.Unmarshal(rec.Body.Bytes(), &revs); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if len(revs) < 1 {
		t.Fatalf("expected revisions for post 13, got %d", len(revs))
	}
	for _, r := range revs {
		if r.Parent != 13 {
			t.Fatalf("parent=%d want 13", r.Parent)
		}
		if r.ID <= 0 || r.Title.Rendered == "" {
			t.Fatalf("bad revision: %+v", r)
		}
	}
	first := revs[0]

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13/revisions/"+strconv.FormatInt(first.ID, 10), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("item status=%d body=%s", rec.Code, rec.Body.String())
	}
	var one rest.Revision
	if err := json.Unmarshal(rec.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if one.ID != first.ID || one.Parent != 13 {
		t.Fatalf("item=%+v", one)
	}
}
