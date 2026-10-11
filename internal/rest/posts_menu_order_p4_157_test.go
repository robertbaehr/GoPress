package rest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.157: menu_order[]= and empty menu_order= → rest_invalid_type.
func TestPagesMenuOrderInvalidFormsP4157(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	for _, path := range []string{
		"/wp-json/wp/v2/pages?menu_order[]=1&per_page=1",
		"/wp-json/wp/v2/pages?menu_order=&per_page=1",
		"/wp-json/wp/v2/pages?menu_order=1.5&per_page=1",
		"/wp-json/wp/v2/pages?menu_order=+1&per_page=1",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		c = c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "menu_order is not of type integer") {
			t.Fatalf("%s: expected integer type error: %s", path, rec.Body.String())
		}
	}
}

// P4.157: create page + update post with menu_order write.
func TestCreatePageAndUpdatePostMenuOrderP4157(t *testing.T) {
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

	createBody := `{"title":"Menu Order Page","content":"x","status":"publish","slug":"menu-order-p4157","menu_order":42}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/pages", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create page status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", page.ID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})
	if page.MenuOrder != 42 {
		t.Fatalf("page menu_order=%d want 42", page.MenuOrder)
	}

	postBody := `{"title":"Menu Order Post","content":"y","status":"draft","slug":"menu-order-post-p4157","menu_order":7}`
	req = httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts", bytes.NewBufferString(postBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create post status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", post.ID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})
	if post.MenuOrder != 7 {
		t.Fatalf("post menu_order=%d want 7", post.MenuOrder)
	}

	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/wp-json/wp/v2/posts/%d", post.ID), bytes.NewBufferString(`{"menu_order":99}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.MenuOrder != 99 {
		t.Fatalf("updated menu_order=%d want 99", updated.MenuOrder)
	}
}
