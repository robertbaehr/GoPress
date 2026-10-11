package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P6.72: lostpassword_post fires with error code + user (or nil) before send.
func TestLostPasswordPostActionP672(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	var gotCode string
	var gotUserID int64 = -1
	c.Hooks.AddAction("lostpassword_post", func(args ...any) {
		if len(args) >= 1 {
			gotCode, _ = args[0].(string)
		}
		gotUserID = 0
		if len(args) >= 2 {
			if u, ok := args[1].(*wp.User); ok && u != nil {
				gotUserID = u.ID
			}
		}
	}, 10, 2)

	form := url.Values{}
	form.Set("user_login", "")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?action=lostpassword", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if gotCode != "empty_username" {
		t.Fatalf("empty code=%q", gotCode)
	}
	if gotUserID != 0 {
		t.Fatalf("empty user=%d", gotUserID)
	}

	admin, err := c.GetUserBy("login", "admin")
	if err != nil || admin == nil {
		t.Fatalf("admin: %v", err)
	}
	form = url.Values{}
	form.Set("user_login", "admin")
	req = httptest.NewRequest(http.MethodPost, "/wp-login.php?action=lostpassword", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if gotCode != "" {
		t.Fatalf("success code=%q", gotCode)
	}
	if gotUserID != admin.ID {
		t.Fatalf("success user=%d want %d", gotUserID, admin.ID)
	}
}
