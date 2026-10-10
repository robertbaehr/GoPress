package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestLinkManagerEnabledAlwaysFalse(t *testing.T) {
	if wp.LinkManagerEnabled() {
		t.Fatal("LinkManagerEnabled must be false")
	}
}

func TestManageLinksDeniedEvenWhenOptionOn(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH:        "/workspace/",
		AuthKey:        "gopress-ref-auth-key-0001-fixed-for-oracle",
		SecureAuthKey:  "gopress-ref-secure-auth-key-0002-fixed",
		LoggedInKey:    "gopress-ref-logged-in-key-0003-fixed",
		NonceKey:       "gopress-ref-nonce-key-0004-fixed-oracle",
		AuthSalt:       "gopress-ref-auth-salt-0005-fixed-oracle",
		SecureAuthSalt: "gopress-ref-secure-auth-salt-0006-fixed",
		LoggedInSalt:   "gopress-ref-logged-in-salt-0007-fixed",
		NonceSalt:      "gopress-ref-nonce-salt-0008-fixed-oracle",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })

	// Attempt to enable in DB; GetOption must still report off.
	_, _ = c.Options.UpdateOption("link_manager_enabled", "1", "")
	if v := c.Options.GetOption("link_manager_enabled", "1"); optionString(v) != "0" {
		t.Fatalf("GetOption link_manager_enabled=%v want 0", v)
	}

	admin, err := c.GetUserBy("login", "admin")
	if err != nil || admin == nil {
		t.Fatalf("admin: %v", err)
	}
	_ = c.Roles.InitUser(admin, c.Meta)
	if c.Roles.UserCan(admin, "manage_links") {
		t.Fatal("manage_links must stay denied (link manager gone)")
	}
}

func optionString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}
