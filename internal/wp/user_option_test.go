package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetUpdateUserOption(t *testing.T) {
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
	key := "edit_post_per_page"
	prev := wp.GetUserOption(c, key, user.ID)
	t.Cleanup(func() {
		if prev == "" {
			_, _ = c.Meta.DeleteMetadata(wp.MetaUser, user.ID, c.DB.TablePrefix+key, "")
		} else {
			_ = wp.UpdateUserOption(c, user.ID, key, prev)
		}
	})

	if err := wp.UpdateUserOption(c, user.ID, key, "7"); err != nil {
		t.Fatalf("UpdateUserOption: %v", err)
	}
	if got := wp.GetUserOption(c, key, user.ID); got != "7" {
		t.Fatalf("GetUserOption=%q want 7", got)
	}
	c.CurrentUser = user
	if got := wp.EditPostsPerPage(c, "post"); got != 7 {
		t.Fatalf("EditPostsPerPage=%d want 7", got)
	}
}

func TestEditPostsPerPageDefault(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()
	c.CurrentUser = nil
	if got := wp.EditPostsPerPage(c, "post"); got != 20 {
		t.Fatalf("default=%d want 20", got)
	}
}
