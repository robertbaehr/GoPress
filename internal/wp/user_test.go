package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func testBootConfig() *config.Config {
	return &config.Config{
		DBName:         "gopress_ref",
		DBUser:         "gopress",
		DBPassword:     "gopress",
		DBHost:         "127.0.0.1:3306",
		DBCharset:      "utf8mb4",
		TablePrefix:    "wp_",
		AuthKey:        "gopress-ref-auth-key-0001-fixed-for-oracle",
		SecureAuthKey:  "gopress-ref-secure-auth-key-0002-fixed",
		LoggedInKey:    "gopress-ref-logged-in-key-0003-fixed",
		NonceKey:       "gopress-ref-nonce-key-0004-fixed-oracle",
		AuthSalt:       "gopress-ref-auth-salt-0005-fixed-oracle",
		SecureAuthSalt: "gopress-ref-secure-auth-salt-0006-fixed",
		LoggedInSalt:   "gopress-ref-logged-in-salt-0007-fixed",
		NonceSalt:      "gopress-ref-nonce-salt-0008-fixed-oracle",
	}
}

func TestGetUserByFixtureAdmin(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	if c.Users == nil || c.Auth == nil {
		t.Fatal("Users/Auth not wired on Boot")
	}

	byID, err := c.Users.GetUserBy("id", 1)
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	if byID == nil || byID.Login != "admin" || byID.Email != "admin@example.com" {
		t.Fatalf("by id: %#v", byID)
	}

	byLogin, err := c.Users.GetUserBy("login", "admin")
	if err != nil || byLogin == nil || byLogin.ID != 1 {
		t.Fatalf("by login: %#v err=%v", byLogin, err)
	}

	byEmail, err := c.Users.GetUserBy("email", "admin@example.com")
	if err != nil || byEmail == nil || byEmail.ID != 1 {
		t.Fatalf("by email: %#v err=%v", byEmail, err)
	}

	if !byID.CheckPassword("password") {
		t.Fatal("CheckPassword('password') expected true for fixture admin")
	}
	if byID.CheckPassword("wrong-password") {
		t.Fatal("CheckPassword('wrong-password') expected false")
	}

	c.CurrentUser = byID
	if c.CurrentUser == nil || c.CurrentUser.ID != 1 {
		t.Fatal("CurrentUser not wired")
	}
}

func TestGetUserByMissing(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	u, err := c.Users.GetUserBy("login", "no-such-user-xyz")
	if err != nil {
		t.Fatal(err)
	}
	if u != nil {
		t.Fatalf("expected nil, got %#v", u)
	}
}

func TestGetUserByUnsupportedField(t *testing.T) {
	u := wp.NewUsers(nil, nil)
	_, err := u.GetUserBy("phone", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}
