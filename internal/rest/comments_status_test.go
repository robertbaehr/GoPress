package rest_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestCommentsStatusHoldForbiddenAnon(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?status=hold", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "rest_forbidden_param") {
		t.Fatalf("expected rest_forbidden_param: %s", rec.Body.String())
	}
}

func TestCommentsStatusHoldAsAdmin(t *testing.T) {
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

	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Comments()+
			" (comment_post_ID, comment_author, comment_author_email, comment_author_url, comment_author_IP, "+
			"comment_date, comment_date_gmt, comment_content, comment_karma, comment_approved, comment_agent, "+
			"comment_type, comment_parent, user_id) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %d, %s, %s, %s, %d, %d)",
		1, "Hold Auth", "hold@example.com", "", "127.0.0.1",
		"2024-06-01 00:00:00", "2024-06-01 00:00:00", "rest status hold target", 0, "0", "test",
		"comment", 0, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := res.LastInsertId()
	t.Cleanup(func() {
		dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.Comments()+" WHERE comment_ID = %d", cid)
		_, _ = c.DB.Exec(dq, dargs...)
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?status=hold&per_page=100", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var comments []rest.Comment
	if err := json.Unmarshal(rec.Body.Bytes(), &comments); err != nil {
		t.Fatalf("json: %v", err)
	}
	found := false
	for _, cm := range comments {
		if cm.ID == cid {
			found = true
			if cm.Status != "hold" {
				t.Fatalf("status=%q want hold", cm.Status)
			}
		}
		if cm.Status != "hold" {
			t.Fatalf("unexpected status %q in hold filter", cm.Status)
		}
	}
	if !found {
		t.Fatalf("hold comment %d not in list (len=%d)", cid, len(comments))
	}

	// Approve via PATCH (status write path).
	patch := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/wp-json/wp/v2/comments/%d", cid),
		strings.NewReader(`{"status":"approved"}`))
	patch.Header.Set("Content-Type", "application/json")
	patch.AddCookie(ck)
	prec := httptest.NewRecorder()
	prc := c.ForRequest(prec, patch)
	(&rest.Server{}).Handle(prec, patch, prc)
	if prec.Code != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", prec.Code, prec.Body.String())
	}
	var approved rest.Comment
	if err := json.Unmarshal(prec.Body.Bytes(), &approved); err != nil {
		t.Fatalf("approve json: %v", err)
	}
	if approved.Status != "approved" {
		t.Fatalf("after approve status=%q", approved.Status)
	}
}

func TestCommentsStatusApproveDefault(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?status=approve&per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var comments []rest.Comment
	if err := json.Unmarshal(rec.Body.Bytes(), &comments); err != nil {
		t.Fatalf("json: %v", err)
	}
	for _, cm := range comments {
		if cm.Status != "approved" {
			t.Fatalf("status=%q", cm.Status)
		}
	}
}
