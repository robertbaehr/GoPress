package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestQuickDraftCreatesPost(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	form := url.Values{}
	form.Set("action", "post-quickdraft-save")
	form.Set("post_title", "Quick Draft Title")
	form.Set("content", "Hello from Quick Draft")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/post.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Draft created successfully") {
		t.Fatalf("expected success notice, got %q", body)
	}
	if !strings.Contains(body, "Quick Draft") {
		t.Fatalf("expected dashboard re-render, got %q", body)
	}

	q, args, err := c.DB.Prepare(
		"SELECT ID, post_status, post_content FROM "+c.DB.Posts()+
			" WHERE post_title = %s AND post_type = %s ORDER BY ID DESC LIMIT 1",
		"Quick Draft Title", "post",
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("draft not found: %v", err)
	}
	status := ""
	switch v := row["post_status"].(type) {
	case string:
		status = v
	case []byte:
		status = string(v)
	}
	if status != "draft" {
		t.Fatalf("status=%q want draft", status)
	}
	content := ""
	switch v := row["post_content"].(type) {
	case string:
		content = v
	case []byte:
		content = string(v)
	}
	if !strings.Contains(content, "<!-- wp:paragraph -->") || !strings.Contains(content, "Hello from Quick Draft") {
		t.Fatalf("content=%q", content)
	}
}

func TestQuickDraftRejectsEmpty(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	form := url.Values{}
	form.Set("action", "post-quickdraft-save")
	form.Set("post_title", "")
	form.Set("content", "")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/post.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "empty title and content") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestQuickDraftFormHasAction(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `value="post-quickdraft-save"`) {
		t.Fatalf("missing quickdraft action: %q", body)
	}
}
