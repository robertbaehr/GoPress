package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPostEditShellDeepen(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/post.php?post=1&action=edit", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`post-type-post`,
		`class="page-title-action"`,
		`Add New`,
		`id="excerpt"`,
		`id="edit-slug-box"`,
		`name="action" value="editpost"`,
		`id="_wpnonce"`,
		// P7.127 block editor chrome
		`block-editor-page`,
		`is-fullscreen-mode`,
		`id="editor"`,
		`edit-post-layout`,
		`__gopressBlockEditorSettings`,
		`pagenow = "post"`,
		`The block editor requires JavaScript`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body", want)
		}
	}
}

func TestPostNewBlockEditorStub(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/post-new.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`post-new-php`,
		`block-editor-page`,
		`id="editor"`,
		`pagenow = "post-new"`,
		`"isNew":true`,
		`id="title"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body", want)
		}
	}
}

func TestPostEditSave(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	orig, err := wp.GetPostByID(c, 1)
	if err != nil || orig == nil {
		t.Fatalf("orig post: %v", err)
	}
	t.Cleanup(func() {
		q, args, perr := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_title = %s, post_content = %s, post_excerpt = %s, post_name = %s, post_status = %s, "+
				"post_modified = %s, post_modified_gmt = %s WHERE ID = %d",
			orig.Title, orig.Content, orig.Excerpt, orig.Name, orig.Status, orig.Modified, orig.ModifiedGMT, orig.ID,
		)
		if perr == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/post.php?post=1&action=edit", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	re := regexp.MustCompile(`name="_wpnonce" value="([^"]+)"`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		t.Fatal("missing _wpnonce")
	}
	form := url.Values{}
	form.Set("_wpnonce", m[1])
	form.Set("post_ID", "1")
	form.Set("action", "editpost")
	form.Set("post_title", "Hello world — edited")
	form.Set("content", "Updated body")
	form.Set("excerpt", "Short excerpt")
	form.Set("post_name", "hello-world")
	form.Set("save", "Update")

	req = httptest.NewRequest(http.MethodPost, "/wp-admin/post.php?post=1&action=edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Post updated.") {
		t.Fatalf("expected update notice: %q", rec.Body.String()[:min(400, len(rec.Body.String()))])
	}
	post, err := wp.GetPostByID(c, 1)
	if err != nil || post == nil {
		t.Fatalf("reload post: %v", err)
	}
	if post.Title != "Hello world — edited" {
		t.Fatalf("title=%q", post.Title)
	}
	if post.Excerpt != "Short excerpt" {
		t.Fatalf("excerpt=%q", post.Excerpt)
	}
}
