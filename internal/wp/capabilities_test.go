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

func TestMapMetaCapEditPostOwnership(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	author, err := c.GetUserBy("login", "author")
	if err != nil || author == nil {
		t.Fatalf("author: %v", err)
	}
	editor, err := c.GetUserBy("login", "editor")
	if err != nil || editor == nil {
		t.Fatalf("editor: %v", err)
	}

	// Find a published post owned by admin (user 1) and one owned by author if any.
	adminPost, err := wp.GetPost(c, wp.QueryArgs{PostType: "post", PostStatus: "publish", PostsPerPage: 1})
	if err != nil || adminPost == nil {
		t.Fatalf("admin post: %v", err)
	}
	// Author cannot edit someone else's published post without edit_others_posts.
	if adminPost.Author != author.ID {
		if c.UserCan(author, "edit_post", adminPost.ID) {
			t.Fatalf("author should not edit_post %d (author=%d)", adminPost.ID, adminPost.Author)
		}
		if c.UserCan(author, "delete_post", adminPost.ID) {
			t.Fatalf("author should not delete_post %d", adminPost.ID)
		}
	}
	// Editor has edit_others_posts and can edit.
	if !c.UserCan(editor, "edit_post", adminPost.ID) {
		t.Fatalf("editor should edit_post %d", adminPost.ID)
	}
	if !c.UserCan(editor, "delete_post", adminPost.ID) {
		t.Fatalf("editor should delete_post %d", adminPost.ID)
	}

	// Author can edit/delete their own draft.
	ownDraft, err := wp.GetPost(c, wp.QueryArgs{
		PostType: "post", PostStatus: "draft", PostsPerPage: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = ownDraft
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostType: "post", PostStatus: "draft", PostsPerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, p := range posts {
		if p.Author == author.ID {
			found = true
			if !c.UserCan(author, "edit_post", p.ID) {
				t.Fatalf("author should edit own draft %d", p.ID)
			}
			if !c.UserCan(author, "delete_post", p.ID) {
				t.Fatalf("author should delete own draft %d", p.ID)
			}
			caps := c.MapMetaCap("edit_post", author.ID, p.ID)
			if len(caps) != 1 || caps[0] != "edit_posts" {
				t.Fatalf("own draft map_meta_cap=%v want [edit_posts]", caps)
			}
			break
		}
	}
	if !found {
		t.Fatal("no draft owned by author in fixture")
	}

	// Missing object ID denies.
	if c.UserCan(author, "edit_post") {
		t.Fatal("edit_post without ID should deny")
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

func TestCurrentUserCan(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	if c.CurrentUserCan("read") {
		t.Fatal("expected false with no CurrentUser")
	}

	admin, err := c.GetUserBy("login", "admin")
	if err != nil || admin == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = admin

	if !c.CurrentUserCan("manage_options") {
		t.Fatal("admin should manage_options")
	}
	if !c.CurrentUserCan("edit_posts") {
		t.Fatal("admin should edit_posts")
	}
	// Self edit_user / app password meta caps map to exist.
	if !c.CurrentUserCan("edit_user", admin.ID) {
		t.Fatal("admin should edit self")
	}
	if !c.CurrentUserCan("list_app_passwords", admin.ID) {
		t.Fatal("admin should list own app passwords")
	}
	if !c.CurrentUserCan("create_app_password", admin.ID) {
		t.Fatal("admin should create own app passwords")
	}
}

func TestMapMetaCapEditUserSelf(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	caps := c.MapMetaCap("edit_user", 1, 1)
	if len(caps) != 1 || caps[0] != "exist" {
		t.Fatalf("self edit_user caps=%v", caps)
	}
	caps = c.MapMetaCap("edit_user", 1, 2)
	if len(caps) != 1 || caps[0] != "edit_users" {
		t.Fatalf("other edit_user caps=%v", caps)
	}
	caps = c.MapMetaCap("list_app_passwords", 1, 1)
	if len(caps) != 1 || caps[0] != "exist" {
		t.Fatalf("list_app_passwords self caps=%v", caps)
	}
}
