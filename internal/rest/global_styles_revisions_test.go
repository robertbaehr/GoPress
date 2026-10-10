package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestGlobalStylesRevisionsRequireAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/global-styles/445/revisions", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGlobalStylesRevisionsListAndItem(t *testing.T) {
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

	content := `{"version":3,"isGlobalStylesUserThemeJSON":true,"settings":{"color":{"palette":{"theme":[]}}},"styles":{"color":{"background":"#111111"}}}`
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status,"+
			" comment_status, ping_status, post_name, to_ping, pinged, post_modified, post_modified_gmt,"+
			" post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count)"+
			" VALUES (%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		user.ID, "2024-06-01 10:00:00", "2024-06-01 10:00:00", content, "Custom Styles", "",
		"inherit", "closed", "closed", "445-revision-v1", "", "",
		"2024-06-01 10:00:00", "2024-06-01 10:00:00", "", 445,
		"http://127.0.0.1:8081/?p=rev-gs", 0, "revision", "", 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatalf("insert revision: %v", err)
	}
	revID, _ := res.LastInsertId()
	t.Cleanup(func() {
		dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", revID)
		_, _ = c.DB.Exec(dq, dargs...)
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/global-styles/445/revisions", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("missing X-WP-Total")
	}
	var revs []rest.GlobalStylesRevision
	if err := json.Unmarshal(rec.Body.Bytes(), &revs); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if len(revs) < 1 {
		t.Fatalf("expected revisions, got %d", len(revs))
	}
	found := false
	for _, r := range revs {
		if r.ID == revID {
			found = true
			if r.Parent != 445 {
				t.Fatalf("parent=%d", r.Parent)
			}
			bg, _ := r.Styles["color"].(map[string]any)
			if bg == nil || bg["background"] != "#111111" {
				t.Fatalf("styles=%#v", r.Styles)
			}
		}
	}
	if !found {
		t.Fatalf("inserted revision %d not in %#v", revID, revs)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/global-styles/445/revisions/"+strconv.FormatInt(revID, 10), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("item status=%d body=%s", rec.Code, rec.Body.String())
	}
	var one rest.GlobalStylesRevision
	if err := json.Unmarshal(rec.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if one.ID != revID || one.Parent != 445 {
		t.Fatalf("item=%+v", one)
	}
}

func TestGlobalStylesRevisionsInvalidParent(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/global-styles/13/revisions", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["code"] != "rest_post_invalid_parent" && body["code"] != "rest_global_styles_not_found" {
		t.Fatalf("code=%v", body["code"])
	}
}
