package wp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetSessionTokenFromContext(t *testing.T) {
	if wp.GetSessionToken(nil) != "" {
		t.Fatal("nil context should be empty")
	}
	c := &wp.Context{}
	if got := wp.GetSessionToken(c); got != "" {
		t.Fatalf("anonymous GetSessionToken=%q want empty", got)
	}
	c.SessionToken = "tok-from-context"
	if got := wp.GetSessionToken(c); got != "tok-from-context" {
		t.Fatalf("GetSessionToken=%q want tok-from-context", got)
	}
}

func TestGetSessionTokenFromLoggedInCookie(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin user: %v", err)
	}
	now := time.Now().Unix()
	set, err := c.SetAuthCookie(user, false, false, now, "")
	if err != nil || set == nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	// Do not call AuthenticateRequest — force cookie parse path.
	c.SessionToken = ""
	got := wp.GetSessionToken(c)
	if got == "" {
		t.Fatal("expected token from logged_in cookie")
	}
	if got != set.Token {
		t.Fatalf("GetSessionToken=%q want %q", got, set.Token)
	}
}

func TestGetSessionTokenAfterAuthenticate(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin user: %v", err)
	}
	now := time.Now().Unix()
	set, err := c.SetAuthCookie(user, false, false, now, "")
	if err != nil || set == nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	c.AuthenticateRequest()
	got := wp.GetSessionToken(c)
	if got != set.Token {
		t.Fatalf("after auth GetSessionToken=%q want %q", got, set.Token)
	}
}
