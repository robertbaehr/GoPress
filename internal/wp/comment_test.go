package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func bootComment(t *testing.T) *wp.Context {
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

func TestListApprovedCommentsHelloWorld(t *testing.T) {
	c := bootComment(t)
	comments, err := wp.ListApprovedComments(c, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) == 0 {
		t.Fatal("expected at least one approved comment on hello-world")
	}
	found := false
	for _, cm := range comments {
		if cm.Approved != "1" {
			t.Fatalf("expected approved only, got %q", cm.Approved)
		}
		if cm.Author == "A WordPress Commenter" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected fixture commenter, got %+v", comments)
	}
}

func TestResolveCommentApproval(t *testing.T) {
	c := bootComment(t)
	// Logged-in users approve when moderation is off.
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	if got := wp.ResolveCommentApproval(c, "admin@example.com", user); got != "1" {
		t.Fatalf("logged-in want 1 got %q", got)
	}
	if got := wp.ResolveCommentApproval(c, "wapuu@wordpress.example", nil); got != "1" {
		t.Fatalf("previously approved email want 1 got %q", got)
	}
	if got := wp.ResolveCommentApproval(c, "brand-new-commenter@example.invalid", nil); got != "0" {
		t.Fatalf("new email with previously_approved want hold, got %q", got)
	}
}
