package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPostsTypeFieldMatchesRouteP4150(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=1&_fields=id,type", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("posts status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil || len(posts) == 0 {
		t.Fatalf("posts decode: %v body=%s", err, rec.Body.String())
	}
	if posts[0]["type"] != "post" {
		t.Fatalf("posts type=%v want post", posts[0]["type"])
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?per_page=1&_fields=id,type", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("pages status=%d body=%s", rec.Code, rec.Body.String())
	}
	var pages []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pages); err != nil || len(pages) == 0 {
		t.Fatalf("pages decode: %v body=%s", err, rec.Body.String())
	}
	if pages[0]["type"] != "page" {
		t.Fatalf("pages type=%v want page", pages[0]["type"])
	}
}

func TestCreatePageForcesTypeP4150(t *testing.T) {
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
	// Body type is ignored; route forces page.
	body := []byte(`{"title":"P4.150 Page","status":"draft","type":"post"}`)
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/pages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["type"] != "page" {
		t.Fatalf("type=%v want page (route wins over body)", out["type"])
	}
	id, _ := out["id"].(float64)
	if id <= 0 {
		t.Fatalf("missing id: %#v", out)
	}
	// Cleanup draft page.
	q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", int64(id))
	if err == nil {
		_, _ = c.DB.Exec(q, args...)
	}
}

func TestCreatePostIgnoresBodyTypeP4150(t *testing.T) {
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
	body := []byte(`{"title":"P4.150 Post","status":"draft","type":"page"}`)
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["type"] != "post" {
		t.Fatalf("type=%v want post", out["type"])
	}
	id, _ := out["id"].(float64)
	if id > 0 {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", int64(id))
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	}
}
