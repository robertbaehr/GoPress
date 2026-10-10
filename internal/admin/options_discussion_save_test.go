package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestOptionsDiscussionSaveViaOptionsPHP(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	keys := []string{
		"default_comment_status", "default_ping_status", "default_pingback_flag",
		"require_name_email", "comment_registration", "close_comments_for_old_posts",
		"close_comments_days_old", "thread_comments", "thread_comments_depth",
		"page_comments", "comments_per_page", "comments_notify", "moderation_notify",
		"comment_moderation", "comment_previously_approved", "comment_max_links",
		"moderation_keys", "disallowed_keys", "show_avatars", "avatar_rating", "avatar_default",
	}
	orig := map[string]any{}
	for _, k := range keys {
		orig[k] = c.Options.GetOption(k, "")
	}
	defer func() {
		for k, v := range orig {
			_, _ = c.Options.UpdateOption(k, v, "yes")
		}
	}()

	form := url.Values{}
	form.Set("option_page", "discussion")
	// Leave default_ping_status unchecked → closed.
	form.Set("default_comment_status", "open")
	form.Set("require_name_email", "1")
	form.Set("thread_comments", "1")
	form.Set("thread_comments_depth", "3")
	form.Set("comments_per_page", "25")
	form.Set("default_pingback_flag", "1")
	form.Set("comment_max_links", "3")
	form.Set("moderation_keys", "spamword\nbadip")
	form.Set("disallowed_keys", "blocked")
	form.Set("show_avatars", "1")
	form.Set("avatar_rating", "PG")
	form.Set("avatar_default", "identicon")
	form.Set("submit", "Save Changes")

	h := &admin.Handler{}
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/options.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options.php") {
		t.Fatal("options.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "options-discussion.php") || !strings.Contains(loc, "settings-updated=true") {
		t.Fatalf("Location=%q", loc)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("default_comment_status", ""))); got != "open" {
		t.Fatalf("default_comment_status=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("default_ping_status", ""))); got != "closed" {
		t.Fatalf("default_ping_status=%v (unchecked)", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("thread_comments_depth", ""))); got != "3" {
		t.Fatalf("thread_comments_depth=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("comments_per_page", ""))); got != "25" {
		t.Fatalf("comments_per_page=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("comment_max_links", ""))); got != "3" {
		t.Fatalf("comment_max_links=%v", got)
	}
	if got := fmt.Sprint(c.Options.GetOption("moderation_keys", "")); !strings.Contains(got, "spamword") {
		t.Fatalf("moderation_keys=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("avatar_rating", ""))); got != "PG" {
		t.Fatalf("avatar_rating=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("avatar_default", ""))); got != "identicon" {
		t.Fatalf("avatar_default=%v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options-discussion.php?settings-updated=true", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-discussion.php") {
		t.Fatal("options-discussion.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Settings saved.") {
		t.Fatalf("expected saved notice: %q", body)
	}
	if !strings.Contains(body, `name="option_page"`) || !strings.Contains(body, `value="discussion"`) {
		t.Fatalf("expected option_page=discussion: %q", body)
	}
	for _, want := range []string{
		`name="default_pingback_flag"`,
		`name="moderation_keys"`,
		`name="disallowed_keys"`,
		`name="avatar_rating"`,
		`name="avatar_default"`,
		`value="identicon"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in discussion form", want)
		}
	}
}
