package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetCurrentUserID(t *testing.T) {
	if wp.GetCurrentUserID(nil) != 0 {
		t.Fatal("nil context should be 0")
	}
	c := &wp.Context{}
	if wp.GetCurrentUserID(c) != 0 {
		t.Fatal("anonymous should be 0")
	}
	if wp.GetCurrentUser(c) != nil {
		t.Fatal("anonymous GetCurrentUser should be nil")
	}

	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	boot, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer boot.DB.Close()
	user, err := boot.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	boot.CurrentUser = user
	if got := wp.GetCurrentUserID(boot); got != user.ID {
		t.Fatalf("GetCurrentUserID=%d want %d", got, user.ID)
	}
	if got := wp.GetCurrentUser(boot); got == nil || got.ID != user.ID {
		t.Fatalf("GetCurrentUser=%v", got)
	}
}
