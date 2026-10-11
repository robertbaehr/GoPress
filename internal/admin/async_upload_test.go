package admin_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestAsyncUpload(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("async-upload", "async-test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("async-bytes")); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/async-upload.php", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, c, "/wp-admin/async-upload.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID       int64  `json:"id"`
			URL      string `json:"url"`
			Filename string `json:"filename"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Data.ID <= 0 {
		t.Fatalf("resp=%#v", resp)
	}
	if !strings.HasPrefix(resp.Data.Filename, "2024/06/") {
		t.Fatalf("filename=%q", resp.Data.Filename)
	}
	t.Cleanup(func() {
		_ = os.Remove("/workspace/wp-content/uploads/" + resp.Data.Filename)
		if c.DB != nil {
			q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", resp.Data.ID)
			_, _ = c.DB.Exec(q, args...)
			q2, a2, _ := c.DB.Prepare("DELETE FROM "+c.DB.PostMeta()+" WHERE post_id = %d", resp.Data.ID)
			_, _ = c.DB.Exec(q2, a2...)
		}
	})
}

func TestAsyncUploadActionAttachment(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("action", "upload-attachment")
	part, err := mw.CreateFormFile("async-upload", "async-ajax.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("ajax-bytes")); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/async-upload.php", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, c, "/wp-admin/async-upload.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff: %v", rec.Header())
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "no-cache") {
		t.Fatalf("missing nocache: %v", rec.Header())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Data.ID <= 0 {
		t.Fatalf("resp=%#v", resp)
	}
	t.Cleanup(func() {
		if c.DB != nil {
			q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", resp.Data.ID)
			_, _ = c.DB.Exec(q, args...)
			q2, a2, _ := c.DB.Prepare("DELETE FROM "+c.DB.PostMeta()+" WHERE post_id = %d", resp.Data.ID)
			_, _ = c.DB.Exec(q2, a2...)
		}
	})
}

func TestAsyncUploadShortAndFetch(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("short", "1")
	part, err := mw.CreateFormFile("async-upload", "async-short.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("short-bytes")); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/async-upload.php", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, c, "/wp-admin/async-upload.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("want text/plain, got %q", rec.Header().Get("Content-Type"))
	}
	idStr := strings.TrimSpace(rec.Body.String())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		t.Fatalf("short body=%q", idStr)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/async-upload.php?attachment_id="+idStr+"&fetch=3", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, c, "/wp-admin/async-upload.php") {
		t.Fatal("fetch not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch status=%d body=%s", rec.Code, rec.Body.String())
	}
	bodyHTML := rec.Body.String()
	if !strings.Contains(bodyHTML, `media-item-wrapper`) || !strings.Contains(bodyHTML, `edit-attachment`) {
		t.Fatalf("fetch html: %q", bodyHTML)
	}
	t.Cleanup(func() {
		if c.DB != nil {
			q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
			_, _ = c.DB.Exec(q, args...)
			q2, a2, _ := c.DB.Prepare("DELETE FROM "+c.DB.PostMeta()+" WHERE post_id = %d", id)
			_, _ = c.DB.Exec(q2, a2...)
		}
	})
}
