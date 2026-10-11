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
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPostsMetaFootnotesReadWrite(t *testing.T) {
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

	createBody := `{"title":"Meta Footnotes Post","content":"Body","status":"publish","slug":"gopress-meta-footnotes","meta":{"footnotes":"[{\"id\":\"a\",\"content\":\"Note A\"}]"}}`
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
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", created.ID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		if c.Meta != nil {
			_, _ = c.Meta.DeleteMetadata(wp.MetaPost, created.ID, "footnotes", "")
		}
	})
	if got, _ := created.Meta["footnotes"].(string); got == "" || got == "[]" {
		t.Fatalf("create meta.footnotes=%v", created.Meta["footnotes"])
	}

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wp-json/wp/v2/posts/%d", created.ID), nil)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d", rec.Code)
	}
	var got rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if fn, _ := got.Meta["footnotes"].(string); fn == "" || !strings.Contains(fn, "Note A") {
		t.Fatalf("GET meta.footnotes=%v", got.Meta["footnotes"])
	}

	updateBody := `{"meta":{"footnotes":""}}`
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/wp-json/wp/v2/posts/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cleared rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &cleared); err != nil {
		t.Fatal(err)
	}
	if fn, _ := cleared.Meta["footnotes"].(string); fn != "" {
		t.Fatalf("cleared meta.footnotes=%q", fn)
	}
}

func TestPostsMetaOmitLeavesFootnotes(t *testing.T) {
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

	createBody := `{"title":"Meta Omit","content":"x","status":"draft","meta":{"footnotes":"keep-me"}}`
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
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", created.ID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		if c.Meta != nil {
			_, _ = c.Meta.DeleteMetadata(wp.MetaPost, created.ID, "footnotes", "")
		}
	})

	updateBody := `{"title":"Meta Omit Updated"}`
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/wp-json/wp/v2/posts/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if fn, _ := updated.Meta["footnotes"].(string); fn != "keep-me" {
		t.Fatalf("omit should leave footnotes: %q", fn)
	}
}
