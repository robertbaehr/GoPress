package rest_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestRESTRevisionDeleteRequiresForce(t *testing.T) {
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

	parentID := int64(13)
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		user.ID, now, now, "rev body", "temp-revision-delete", "", "inherit",
		"closed", "closed", "", strconv.FormatInt(parentID, 10)+"-revision-v1", "", "", now, now,
		"", parentID, "http://example.test/?p="+strconv.FormatInt(parentID, 10), 0, "revision", "", 0,
	)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	revID, err := res.LastInsertId()
	if err != nil || revID <= 0 {
		t.Fatalf("LastInsertId: %v id=%d", err, revID)
	}
	t.Cleanup(func() {
		dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", revID)
		_, _ = c.DB.Exec(dq, dargs...)
	})

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/posts/%d/revisions/%d", parentID, revID), nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 without force, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/posts/%d/revisions/%d?force=true", parentID, revID), nil)
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
	if prev == nil || int64(prev["id"].(float64)) != revID {
		t.Fatalf("previous: %#v", resp["previous"])
	}

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wp-json/wp/v2/posts/%d/revisions/%d", parentID, revID), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRESTRevisionDeleteRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/posts/13/revisions/1?force=true", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
