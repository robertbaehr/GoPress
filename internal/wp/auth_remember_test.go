package wp_test

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestRememberMeSetsLongerCookie(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	now := time.Now().Unix()
	remember, err := c.Login("admin", "password", true, false, now)
	if err != nil || remember == nil {
		t.Fatalf("remember login: %v %#v", err, remember)
	}
	// WP: expiration = now + 14 days; cookie Expire = expiration + 12 hours.
	wantExp := now + 14*24*3600
	if remember.Expiration != wantExp {
		t.Fatalf("remember Expiration=%d want %d", remember.Expiration, wantExp)
	}
	if remember.Expire != wantExp+12*3600 {
		t.Fatalf("remember Expire=%d want %d", remember.Expire, wantExp+12*3600)
	}

	rec := httptest.NewRecorder()
	remember.WriteAuthCookies(rec, "/")
	var sawMaxAge bool
	for _, ck := range rec.Result().Cookies() {
		if ck.MaxAge < 13*24*3600 {
			t.Fatalf("cookie %s MaxAge=%d too short for rememberme", ck.Name, ck.MaxAge)
		}
		if ck.Expires.IsZero() || ck.Expires.Before(time.Now().Add(13*24*time.Hour)) {
			t.Fatalf("cookie %s Expires=%v not long-lived", ck.Name, ck.Expires)
		}
		sawMaxAge = true
	}
	if !sawMaxAge {
		t.Fatal("no cookies written")
	}

	session, err := c.Login("admin", "password", false, false, now)
	if err != nil || session == nil {
		t.Fatalf("session login: %v", err)
	}
	if session.Expire != 0 {
		t.Fatalf("session Expire should be 0, got %d", session.Expire)
	}
	rec = httptest.NewRecorder()
	session.WriteAuthCookies(rec, "/")
	for _, ck := range rec.Result().Cookies() {
		if ck.MaxAge != 0 {
			t.Fatalf("session cookie %s MaxAge=%d want 0", ck.Name, ck.MaxAge)
		}
		if !ck.Expires.IsZero() {
			t.Fatalf("session cookie %s should be session cookie, Expires=%v", ck.Name, ck.Expires)
		}
	}
}
