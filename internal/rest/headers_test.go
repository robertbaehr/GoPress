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

func TestIndexAllowAndAPILink(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8082/wp-json/", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Fatalf("Allow=%q", got)
	}
	link := rec.Header().Get("Link")
	if !strings.Contains(link, `/wp-json/>; rel="https://api.w.org/"`) {
		t.Fatalf("Link=%q", link)
	}
}

func TestPostsCollectionHeaders(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8082/wp-json/wp/v2/posts?per_page=2", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type=%q", ct)
	}
	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Fatalf("Allow=%q", got)
	}
	total := rec.Header().Get("X-WP-Total")
	totalPages := rec.Header().Get("X-WP-TotalPages")
	if total == "" || totalPages == "" {
		t.Fatalf("missing totals: total=%q pages=%q", total, totalPages)
	}
	n, _ := strconv.Atoi(total)
	if n < 2 {
		t.Fatalf("X-WP-Total=%s", total)
	}
	link := rec.Header().Get("Link")
	if !strings.Contains(link, `rel="next"`) {
		t.Fatalf("Link=%q", link)
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("len=%d", len(posts))
	}
}

func TestPagesCollectionHeaders(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8082/wp-json/wp/v2/pages?per_page=2&page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("missing X-WP-Total")
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("ct=%q", rec.Header().Get("Content-Type"))
	}
}

func TestPostEditContextRawFields(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?context=edit", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatal(err)
	}
	if post.Title.Raw == "" || post.Title.Rendered == "" {
		t.Fatalf("title raw/rendered missing: %+v", post.Title)
	}
	if post.Content.Raw == "" {
		t.Fatalf("content.raw empty: %+v", post.Content)
	}
}

func TestPostEditContextAnonNoRaw(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?context=edit", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatal(err)
	}
	if post.Title.Raw != "" || post.Content.Raw != "" {
		t.Fatalf("anon should not see raw fields: title=%+v content=%+v", post.Title, post.Content)
	}
}
