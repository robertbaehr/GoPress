package login_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.68: wp_logout fires with the logged-out user ID after cookies clear.
func TestWPLogoutActionP668(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	var gotID int64 = -1
	var gotNilUser bool
	c.Hooks.AddAction("wp_logout", func(args ...any) {
		if len(args) != 1 {
			t.Fatalf("args=%v", args)
		}
		id, ok := args[0].(int64)
		if !ok {
			t.Fatalf("arg type %T", args[0])
		}
		gotID = id
		gotNilUser = c.CurrentUser == nil
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=logout", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	if gotID != user.ID {
		t.Fatalf("wp_logout user_id=%d want %d", gotID, user.ID)
	}
	if !gotNilUser {
		t.Fatal("expected CurrentUser cleared before wp_logout")
	}
}

// P6.68: anonymous logout still fires wp_logout with user_id 0.
func TestWPLogoutActionAnonymousP668(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	var gotID int64 = -1
	c.Hooks.AddAction("wp_logout", func(args ...any) {
		if len(args) == 1 {
			if id, ok := args[0].(int64); ok {
				gotID = id
			}
		}
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=logout", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if gotID != 0 {
		t.Fatalf("anon wp_logout user_id=%d", gotID)
	}
}
