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
		"default_comment_status", "default_ping_status", "require_name_email",
		"comment_registration", "thread_comments", "thread_comments_depth",
		"comments_per_page", "show_avatars",
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
	form.Set("show_avatars", "1")
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
}
