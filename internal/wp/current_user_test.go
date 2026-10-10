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
	// P2.84: ID <= 0 is treated as anonymous even if CurrentUser is set.
	c.CurrentUser = &wp.User{ID: 0, Login: "ghost"}
	if wp.GetCurrentUserID(c) != 0 || wp.GetCurrentUser(c) != nil {
		t.Fatal("ID 0 user must be anonymous")
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

func TestSetCurrentUser(t *testing.T) {
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

	var fired []int64
	boot.Hooks.AddAction("set_current_user", func(args ...any) {
		if len(args) > 0 {
			if id, ok := args[0].(int64); ok {
				fired = append(fired, id)
			}
		}
	}, 10, 1)

	got := wp.SetCurrentUser(boot, user.ID)
	if got == nil || got.ID != user.ID {
		t.Fatalf("SetCurrentUser=%v", got)
	}
	if wp.GetCurrentUserID(boot) != user.ID {
		t.Fatalf("GetCurrentUserID after set=%d", wp.GetCurrentUserID(boot))
	}
	if len(fired) != 1 || fired[0] != user.ID {
		t.Fatalf("set_current_user fired=%v", fired)
	}

	// Same ID short-circuits without a second action.
	_ = wp.SetCurrentUser(boot, user.ID)
	if len(fired) != 1 {
		t.Fatalf("expected short-circuit, fired=%v", fired)
	}

	cleared := wp.SetCurrentUser(boot, 0)
	if cleared != nil || wp.GetCurrentUser(boot) != nil {
		t.Fatal("SetCurrentUser(0) should clear")
	}
	if len(fired) != 2 || fired[1] != 0 {
		t.Fatalf("clear action fired=%v", fired)
	}

	missing := wp.SetCurrentUser(boot, 999999999)
	if missing != nil || wp.GetCurrentUserID(boot) != 0 {
		t.Fatal("missing user should clear current")
	}
}
