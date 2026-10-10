package wp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestForceSSLAdminToggle(t *testing.T) {
	off := false
	on := true
	_ = wp.ForceSSLAdmin(&off)
	if wp.ForceSSLAdmin(nil) {
		t.Fatal("expected false after off")
	}
	prev := wp.ForceSSLAdmin(&on)
	if prev {
		t.Fatal("prev should be false")
	}
	if !wp.ForceSSLAdmin(nil) {
		t.Fatal("expected true")
	}
	_ = wp.ForceSSLAdmin(&off)
}

func TestApplyForceSSLAdminFromConfig(t *testing.T) {
	off := false
	_ = wp.ForceSSLAdmin(&off)
	wp.ApplyForceSSLAdminFromConfig(&config.Config{Extra: map[string]string{"FORCE_SSL_ADMIN": "true"}})
	if !wp.ForceSSLAdmin(nil) {
		t.Fatal("expected FORCE_SSL_ADMIN true")
	}
	_ = wp.ForceSSLAdmin(&off)
}

func TestMaybeRedirectLoginToHTTPS(t *testing.T) {
	off := false
	on := true
	_ = wp.ForceSSLAdmin(&on)
	t.Cleanup(func() { _ = wp.ForceSSLAdmin(&off) })

	c := &wp.Context{Hooks: wp.NewHooks()}
	req := httptest.NewRequest(http.MethodGet, "http://example.test/wp-login.php?redirect_to=%2Fwp-admin%2F", nil)
	req.Host = "example.test"
	c.Req = req
	rec := httptest.NewRecorder()
	if !wp.MaybeRedirectLoginToHTTPS(rec, req, c) {
		t.Fatal("expected redirect")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "https://example.test/wp-login.php?redirect_to=%2Fwp-admin%2F" {
		t.Fatalf("Location=%q", loc)
	}

	// Already SSL → no redirect.
	req2 := httptest.NewRequest(http.MethodGet, "https://example.test/wp-login.php", nil)
	req2.Header.Set("X-Forwarded-Proto", "https")
	c2 := &wp.Context{Hooks: wp.NewHooks(), Req: req2}
	rec2 := httptest.NewRecorder()
	if wp.MaybeRedirectLoginToHTTPS(rec2, req2, c2) {
		t.Fatal("ssl request should not redirect")
	}
}
