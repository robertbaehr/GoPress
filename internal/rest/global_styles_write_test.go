package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGlobalStylesUpdateRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/global-styles/445",
		strings.NewReader(`{"styles":{"color":{"background":"#ffffff"}}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGlobalStylesUpdate(t *testing.T) {
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

	orig, err := wp.GetPost(c, wp.QueryArgs{P: 445, PostType: "wp_global_styles", PostStatus: "publish"})
	if err != nil || orig == nil {
		t.Fatalf("fixture global styles: %v", err)
	}
	t.Cleanup(func() {
		restoreGlobalStylesPost(t, c, orig)
	})

	payload := `{"styles":{"color":{"background":"#abcdef","text":"#111111"}},"settings":{"color":{"custom":false}},"title":{"raw":"GoPress Test Styles"}}`
	req := httptest.NewRequest(http.MethodPut, "/wp-json/wp/v2/global-styles/445", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var item rest.GlobalStylesItem
	if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.Title["raw"] != "GoPress Test Styles" {
		t.Fatalf("title=%#v", item.Title)
	}
	color, _ := item.Styles["color"].(map[string]any)
	if color == nil || color["background"] != "#abcdef" {
		t.Fatalf("styles=%#v", item.Styles)
	}
	if item.Settings == nil {
		t.Fatalf("missing settings")
	}
}

func restoreGlobalStylesPost(t *testing.T, c *wp.Context, orig *wp.Post) {
	t.Helper()
	if c == nil || c.DB == nil || orig == nil {
		return
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_title = %s, post_content = %s WHERE ID = %d AND post_type = %s",
		orig.Title, orig.Content, orig.ID, "wp_global_styles",
	)
	if err != nil {
		t.Errorf("restore prepare: %v", err)
		return
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		t.Errorf("restore exec: %v", err)
	}
}
