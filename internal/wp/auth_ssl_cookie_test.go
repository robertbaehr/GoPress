package wp_test

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSecureAuthCookieWhenHTTPS(t *testing.T) {
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
		t.Fatalf("admin: %v", err)
	}

	secure, err := c.SetAuthCookie(user, false, true, 0, "")
	if err != nil || secure == nil {
		t.Fatalf("secure SetAuthCookie: %v", err)
	}
	if secure.Scheme != "secure_auth" {
		t.Fatalf("scheme=%q want secure_auth", secure.Scheme)
	}
	if !strings.HasPrefix(secure.AuthName, "wordpress_sec_") {
		t.Fatalf("AuthName=%q want wordpress_sec_*", secure.AuthName)
	}

	rec := httptest.NewRecorder()
	secure.WriteAuthCookies(rec, "/")
	var sawAuth, sawLoggedIn bool
	for _, ck := range rec.Result().Cookies() {
		switch {
		case strings.HasPrefix(ck.Name, "wordpress_sec_"):
			sawAuth = true
			if !ck.Secure {
				t.Fatalf("secure_auth cookie %s must set Secure", ck.Name)
			}
			if !ck.HttpOnly {
				t.Fatalf("secure_auth cookie %s must be HttpOnly", ck.Name)
			}
		case strings.HasPrefix(ck.Name, "wordpress_logged_in_"):
			sawLoggedIn = true
			// Fixture home is http:// — logged_in Secure follows home scheme.
			if ck.Secure != secure.SecureLoggedIn {
				t.Fatalf("logged_in Secure=%v want %v", ck.Secure, secure.SecureLoggedIn)
			}
		}
	}
	if !sawAuth || !sawLoggedIn {
		t.Fatalf("missing cookies auth=%v logged_in=%v", sawAuth, sawLoggedIn)
	}

	plain, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil || plain == nil {
		t.Fatalf("plain SetAuthCookie: %v", err)
	}
	if plain.Scheme != "auth" {
		t.Fatalf("scheme=%q want auth", plain.Scheme)
	}
	if strings.HasPrefix(plain.AuthName, "wordpress_sec_") {
		t.Fatalf("non-secure auth must not use wordpress_sec_: %q", plain.AuthName)
	}
	rec2 := httptest.NewRecorder()
	plain.WriteAuthCookies(rec2, "/")
	for _, ck := range rec2.Result().Cookies() {
		if ck.Secure {
			t.Fatalf("non-HTTPS auth cookie %s unexpectedly Secure", ck.Name)
		}
	}
}

func TestSecureLoggedInWhenHomeHTTPS(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	prevHome := c.Options.GetOption("home", "")
	if _, err := c.Options.UpdateOption("home", "https://example.test", "yes"); err != nil {
		t.Fatalf("set home: %v", err)
	}
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("home", prevHome, "yes")
	})

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, true, 0, "")
	if err != nil || set == nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	if !set.SecureLoggedIn {
		t.Fatal("expected SecureLoggedIn when home is https")
	}
	rec := httptest.NewRecorder()
	set.WriteAuthCookies(rec, "/")
	for _, ck := range rec.Result().Cookies() {
		if strings.HasPrefix(ck.Name, "wordpress_logged_in_") && !ck.Secure {
			t.Fatalf("logged_in cookie must be Secure when home is https")
		}
	}
}
