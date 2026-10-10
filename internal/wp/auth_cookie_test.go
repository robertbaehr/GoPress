package wp_test

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestParseAndValidateAuthCookieGolden(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	g := loadAuthGolden(t)
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	token := g.AuthCookie.Token
	if err := c.StoreSessionToken(1, token, g.AuthCookie.Expiration, time.Now().Unix()); err != nil {
		t.Fatalf("StoreSessionToken: %v", err)
	}

	matched, total := 0, 0
	for scheme, cookie := range g.AuthCookie.Cookies {
		total++
		parts, err := wp.ParseAuthCookie(cookie, scheme)
		if err != nil {
			t.Errorf("parse %s: %v", scheme, err)
			continue
		}
		if parts.Username != "admin" || parts.Token != token || parts.Expiration != g.AuthCookie.Expiration {
			t.Errorf("parse %s: %#v", scheme, parts)
			continue
		}
		user, err := c.ValidateAuthCookie(cookie, scheme, g.AuthCookie.Expiration-10, false)
		if err != nil {
			t.Errorf("validate %s: %v", scheme, err)
			continue
		}
		if user == nil || user.ID != 1 {
			t.Errorf("validate %s: user=%v", scheme, user)
			continue
		}
		matched++
	}
	t.Logf("golden auth cookie validate: %d/%d match", matched, total)
	if matched != total {
		t.Fatalf("incomplete: %d/%d", matched, total)
	}
}

func TestLoginSetsAuthCookie(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	now := time.Now().Unix()
	set, err := c.Login("admin", "password", true, false, now)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if set == nil || set.AuthValue == "" || set.LoggedInValue == "" || set.Token == "" {
		t.Fatalf("incomplete cookie set: %#v", set)
	}
	if c.CurrentUser == nil || c.CurrentUser.Login != "admin" {
		t.Fatalf("CurrentUser: %#v", c.CurrentUser)
	}

	user, err := c.ValidateAuthCookie(set.AuthValue, "auth", now+60, false)
	if err != nil || user == nil || user.ID != 1 {
		t.Fatalf("Validate after Login: user=%v err=%v", user, err)
	}
	user, err = c.ValidateAuthCookie(set.LoggedInValue, "logged_in", now+60, false)
	if err != nil || user == nil || user.ID != 1 {
		t.Fatalf("Validate logged_in: user=%v err=%v", user, err)
	}

	rec := httptest.NewRecorder()
	set.WriteAuthCookies(rec, "/")
	cookies := rec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("expected cookies written, got %d", len(cookies))
	}

	if _, err := c.Login("admin", "wrong", false, false, now); err == nil {
		t.Fatal("expected invalid credentials")
	}
}

func TestParseAuthCookieMalformed(t *testing.T) {
	if _, err := wp.ParseAuthCookie("not-a-cookie", "auth"); err == nil {
		t.Fatal("expected error")
	}
}
