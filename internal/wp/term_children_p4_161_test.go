package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func bootTermChildrenOrSkip(t *testing.T) *wp.Context {
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

// P4.161: ExpandTermIDsWithChildren includes descendants only when flagged.
func TestExpandTermIDsWithChildrenP4161(t *testing.T) {
	c := bootTermChildrenOrSkip(t)
	plain := wp.ExpandTermIDsWithChildren(c, "category", []int64{2}, false)
	if len(plain) != 1 || plain[0] != 2 {
		t.Fatalf("no-expand: %v", plain)
	}
	expanded := wp.ExpandTermIDsWithChildren(c, "category", []int64{2}, true)
	seen := map[int64]bool{}
	for _, id := range expanded {
		seen[id] = true
	}
	if !seen[2] || !seen[3] || !seen[4] {
		t.Fatalf("expand Root: %v", expanded)
	}
}
