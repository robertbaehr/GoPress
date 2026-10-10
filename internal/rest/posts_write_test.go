package rest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestCreatePostRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	body := `{"title":"Nope","content":"x","status":"draft"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateUpdateDeletePost(t *testing.T) {
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

	createBody := `{"title":"GoPress Write Post","content":"Body from REST write","status":"publish","slug":"gopress-write-post"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 || created.Slug != "gopress-write-post" || created.Status != "publish" {
		t.Fatalf("created: %+v", created)
	}
	if created.Title.Rendered != "GoPress Write Post" {
		t.Fatalf("title=%q", created.Title.Rendered)
	}
	id := created.ID
	// Register after Close cleanup so LIFO deletes while DB is still open.
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})

	updateBody := `{"title":"GoPress Write Post Updated","content":"Updated body"}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/wp-json/wp/v2/posts/%d", id), bytes.NewBufferString(updateBody))
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
	if updated.Title.Rendered != "GoPress Write Post Updated" {
		t.Fatalf("updated title=%q", updated.Title.Rendered)
	}
	if updated.Content.Rendered != "Updated body" {
		t.Fatalf("content=%q", updated.Content.Rendered)
	}
	// P4.48: updating revisioned fields creates a revision child.
	rq, rargs, err := c.DB.Prepare(
		"SELECT ID, post_name, post_parent, post_type, post_title FROM "+c.DB.Posts()+
			" WHERE post_parent = %d AND post_type = %s AND post_name = %s ORDER BY ID DESC LIMIT 1",
		id, "revision", fmt.Sprintf("%d-revision-v1", id),
	)
	if err != nil {
		t.Fatal(err)
	}
	revRow, err := c.DB.GetRow(rq, rargs...)
	if err != nil || revRow == nil {
		t.Fatalf("expected revision after update: %v", err)
	}
	revID, _ := strconv.ParseInt(fmt.Sprint(revRow["ID"]), 10, 64)
	t.Cleanup(func() {
		dq, dargs, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", revID)
		if err == nil {
			_, _ = c.DB.Exec(dq, dargs...)
		}
	})
	if fmt.Sprint(revRow["post_title"]) != "GoPress Write Post Updated" {
		t.Fatalf("revision title=%v", revRow["post_title"])
	}

	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/posts/%d", id), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var del map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &del); err != nil {
		t.Fatal(err)
	}
	if del["deleted"] != true {
		t.Fatalf("deleted payload: %v", del)
	}
	// Confirm trash status in DB.
	q, args, err := c.DB.Prepare("SELECT post_status FROM "+c.DB.Posts()+" WHERE ID = %d", id)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("row: %v", err)
	}
	if fmt.Sprint(row["post_status"]) != "trash" {
		t.Fatalf("status=%v", row["post_status"])
	}
}

func TestPatchPost(t *testing.T) {
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

	createBody := `{"title":"Patch Me","content":"orig","status":"draft"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/posts", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.Post
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	id := created.ID
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})

	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/wp-json/wp/v2/posts/%d", id), bytes.NewBufferString(`{"status":"publish"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	var patched rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &patched); err != nil {
		t.Fatal(err)
	}
	if patched.Status != "publish" || patched.Title.Rendered != "Patch Me" {
		t.Fatalf("patched: %+v", patched)
	}
}
