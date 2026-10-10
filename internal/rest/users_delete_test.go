package rest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestRESTUserDeleteRequiresForceAndReassigns(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	admin, err := c.GetUserBy("login", "admin")
	if err != nil || admin == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(admin, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	login := fmt.Sprintf("del_user_%d", time.Now().UnixNano()%1_000_000)
	createBody := fmt.Sprintf(
		`{"username":%q,"email":%q,"password":"SecretPass1!","roles":["author"]}`,
		login, login+"@example.com",
	)
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/users", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created rest.User
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created.ID
	t.Cleanup(func() {
		_ = c.DeleteUser(id, nil)
	})

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		id, now, now, "body", "reassign-me", "", "publish",
		"closed", "closed", "", "reassign-me-"+login, "", "", now, now,
		"", 0, "http://example.test/reassign-me", 0, "post", "", 0,
	)
	if err != nil {
		t.Fatalf("prepare post: %v", err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}
	postID, _ := res.LastInsertId()
	t.Cleanup(func() {
		dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", postID)
		_, _ = c.DB.Exec(dq, dargs...)
	})

	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/users/%d", id), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 without force, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/wp-json/wp/v2/users/%d?force=true&reassign=%d", id, admin.ID), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["deleted"] != true {
		t.Fatalf("expected deleted=true: %#v", resp)
	}
	prev, _ := resp["previous"].(map[string]any)
	if prev == nil || int64(prev["id"].(float64)) != id {
		t.Fatalf("previous: %#v", resp["previous"])
	}

	sq, sargs, err := c.DB.Prepare("SELECT post_author AS a FROM "+c.DB.Posts()+" WHERE ID = %d", postID)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(sq, sargs...)
	if err != nil || row == nil {
		t.Fatalf("post reload: %v", err)
	}
	author := fmt.Sprint(row["a"])
	if author != fmt.Sprint(admin.ID) {
		t.Fatalf("post_author=%q want %d", author, admin.ID)
	}

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wp-json/wp/v2/users/%d", id), nil)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}

func TestRESTUserDeleteRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/users/2?force=true", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
