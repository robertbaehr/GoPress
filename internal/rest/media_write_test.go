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

func TestRESTMediaPatchTitleAndAlt(t *testing.T) {
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

	const id int64 = 5 // sample-photo-a
	body := `{"title":"Patched Photo Title","alt_text":"alt from patch"}`
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/wp-json/wp/v2/media/%d", id), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var m rest.Media
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.Title.Rendered != "Patched Photo Title" {
		t.Fatalf("title=%q", m.Title.Rendered)
	}
	if m.AltText != "alt from patch" {
		t.Fatalf("alt_text=%q", m.AltText)
	}
	t.Cleanup(func() {
		q, args, _ := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_title = %s WHERE ID = %d",
			"sample-photo-a", id,
		)
		_, _ = c.DB.Exec(q, args...)
		if c.Meta != nil {
			_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaPost, id, "_wp_attachment_image_alt", "", "")
		}
	})
}

func TestRESTMediaPatchRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodPatch, "/wp-json/wp/v2/media/5", strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRESTMediaPatchDescriptionAndPost(t *testing.T) {
	// P4.116: description + post (parent) on PATCH, without rewriting title.
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

	const id int64 = 5
	var prevTitle, prevContent string
	var prevParent int64
	q, args, _ := c.DB.Prepare(
		"SELECT post_title, post_content, post_parent FROM "+c.DB.Posts()+" WHERE ID = %d", id,
	)
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("load media: %v", err)
	}
	prevTitle = fmt.Sprint(row["post_title"])
	prevContent = fmt.Sprint(row["post_content"])
	fmt.Sscanf(fmt.Sprint(row["post_parent"]), "%d", &prevParent)
	t.Cleanup(func() {
		q, args, _ := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_title = %s, post_content = %s, post_parent = %d WHERE ID = %d",
			prevTitle, prevContent, prevParent, id,
		)
		_, _ = c.DB.Exec(q, args...)
	})

	body := `{"description":"Patched description body","post":1}`
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/wp-json/wp/v2/media/%d", id), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var m rest.Media
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.Title.Rendered != prevTitle && m.Title.Rendered != "" {
		// Title must be unchanged when omitted from body.
		q2, args2, _ := c.DB.Prepare("SELECT post_title FROM "+c.DB.Posts()+" WHERE ID = %d", id)
		row2, _ := c.DB.GetRow(q2, args2...)
		if got := fmt.Sprint(row2["post_title"]); got != prevTitle {
			t.Fatalf("title clobbered: got %q want %q", got, prevTitle)
		}
	}
	if m.Post == nil || *m.Post != 1 {
		t.Fatalf("post=%v want 1", m.Post)
	}
	q3, args3, _ := c.DB.Prepare("SELECT post_content FROM "+c.DB.Posts()+" WHERE ID = %d", id)
	row3, _ := c.DB.GetRow(q3, args3...)
	if got := fmt.Sprint(row3["post_content"]); got != "Patched description body" {
		t.Fatalf("description content=%q", got)
	}
}
