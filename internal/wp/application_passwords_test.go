package wp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func bootAppPass(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	return c
}

func TestCheckApplicationPassword(t *testing.T) {
	c := bootAppPass(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	plain := "abcdEFGHijklMNOPqrstUVWX"
	hash := wp.FastHash(plain)
	prev := c.Meta.GetMetadata(wp.MetaUser, user.ID, "_application_passwords", true)
	if err := wp.SetApplicationPasswordsForTest(c, user.ID, "", "Unit Test", hash); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "_application_passwords", prev, "")
	})

	if !wp.CheckApplicationPassword(c, user.ID, plain) {
		t.Fatal("expected match for plaintext")
	}
	// Spaces/dashes stripped like WP.
	grouped := "abcd EFGH ijkl MNOP qrst UVWX"
	if !wp.CheckApplicationPassword(c, user.ID, grouped) {
		t.Fatal("expected match with spaces")
	}
	if wp.CheckApplicationPassword(c, user.ID, "wrongpasswordwrongpasswor") {
		t.Fatal("expected reject")
	}
}

func TestAuthenticateApplicationPasswordBasic(t *testing.T) {
	c := bootAppPass(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	plain := "zzzzYYYYXXXXWWWWVVVVuuuu"
	hash := wp.FastHash(plain)
	prev := c.Meta.GetMetadata(wp.MetaUser, user.ID, "_application_passwords", true)
	if err := wp.SetApplicationPasswordsForTest(c, user.ID, "", "Auth Test", hash); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "_application_passwords", prev, "")
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/users/me", nil)
	req.SetBasicAuth("admin", plain)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	rc.AuthenticateApplicationPassword()
	if rc.CurrentUser == nil || rc.CurrentUser.ID != 1 {
		t.Fatalf("expected admin, got %+v", rc.CurrentUser)
	}
}
