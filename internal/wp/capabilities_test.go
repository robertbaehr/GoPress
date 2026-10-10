package wp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

type capsGolden struct {
	Role         string          `json:"role"`
	UserID       int64           `json:"user_id"`
	Login        string          `json:"login"`
	Capabilities map[string]bool `json:"capabilities"`
}

func loadCapsGolden(t *testing.T, name string) capsGolden {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "tests", "golden", "caps", name+".json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var g capsGolden
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("json: %v", err)
	}
	return g
}

func TestRolesLoadFromOptions(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	admin := c.Roles.Role("administrator")
	if admin == nil || !admin.Capabilities["manage_options"] {
		t.Fatalf("administrator: %#v", admin)
	}
	sub := c.Roles.Role("subscriber")
	if sub == nil || !sub.Capabilities["read"] || sub.Capabilities["edit_posts"] {
		t.Fatalf("subscriber: %#v", sub)
	}
}

func TestUserCanGoldenCaps(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	roles := []string{"admin", "editor", "author", "contributor", "subscriber"}
	matched, total := 0, 0
	for _, slug := range roles {
		g := loadCapsGolden(t, slug)
		user, err := c.GetUserBy("login", g.Login)
		if err != nil || user == nil {
			t.Fatalf("%s: GetUserBy: user=%v err=%v", slug, user, err)
		}
		if user.ID != g.UserID {
			t.Fatalf("%s: user_id want %d got %d", slug, g.UserID, user.ID)
		}
		for cap, want := range g.Capabilities {
			total++
			got := c.UserCan(user, cap)
			if got == want {
				matched++
			} else {
				t.Errorf("%s cap %s: want %v got %v", slug, cap, want, got)
			}
		}
	}
	rate := 100 * float64(matched) / float64(total)
	t.Logf("golden caps: %d/%d match (%.1f%%)", matched, total, rate)
	if matched != total {
		t.Fatalf("incomplete caps match: %d/%d", matched, total)
	}
}

func TestPHPUnserializeRolesRoundTrip(t *testing.T) {
	// Smoke: parse a tiny PHP array like wp_capabilities.
	// Exported via Roles path; exercise through InitUser on a synthetic meta cache.
	cache := wp.NewObjectCache()
	cache.Set("1", map[string][]string{
		"wp_capabilities": {`a:1:{s:10:"subscriber";b:1;}`},
	}, "user_meta")
	meta := wp.NewMeta(nil, cache)
	optsCache := wp.NewObjectCache()
	optsCache.Set("alloptions", map[string]string{
		"wp_user_roles":        `a:1:{s:10:"subscriber";a:2:{s:4:"name";s:10:"Subscriber";s:12:"capabilities";a:2:{s:4:"read";b:1;s:7:"level_0";b:1;}}}`,
		"link_manager_enabled": "0",
	}, "options")
	roles := wp.NewRoles(wp.NewOptions(nil, optsCache))
	if err := roles.Load(); err != nil {
		t.Fatal(err)
	}
	user := &wp.User{ID: 1, Login: "sub"}
	if err := roles.InitUser(user, meta); err != nil {
		t.Fatal(err)
	}
	if !roles.UserCan(user, "read") {
		t.Fatal("expected read")
	}
	if roles.UserCan(user, "edit_posts") {
		t.Fatal("did not expect edit_posts")
	}
	if roles.UserCan(user, "manage_links") {
		t.Fatal("manage_links should be denied when link manager off")
	}
	if roles.UserCan(user, "install_plugins") {
		t.Fatal("install_plugins should be denied")
	}
}
