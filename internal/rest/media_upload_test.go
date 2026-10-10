package rest_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestRESTMediaUpload(t *testing.T) {
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

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "rest-upload.png")
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/media", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var m rest.Media
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.ID <= 0 || m.Type != "attachment" {
		t.Fatalf("got %#v", m)
	}
	if !strings.Contains(m.SourceURL, "/wp-content/uploads/2024/06/") {
		t.Fatalf("source_url=%q", m.SourceURL)
	}
	t.Cleanup(func() {
		if c.DB != nil {
			q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", m.ID)
			_, _ = c.DB.Exec(q, args...)
			q2, a2, _ := c.DB.Prepare("DELETE FROM "+c.DB.PostMeta()+" WHERE post_id = %d", m.ID)
			_, _ = c.DB.Exec(q2, a2...)
		}
		rel := strings.TrimPrefix(m.SourceURL, strings.TrimRight(
			func() string {
				if c.Options == nil {
					return ""
				}
				v, _ := c.Options.GetOption("home", "").(string)
				return strings.TrimRight(v, "/") + "/wp-content/uploads/"
			}(), "/"))
		_ = rel
		_ = os.Remove("/workspace/wp-content/uploads/2024/06/rest-upload.png")
		_ = os.Remove("/workspace/wp-content/uploads/2024/06/rest-upload-1.png")
	})
}
