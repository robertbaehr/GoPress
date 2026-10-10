package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func testBootQuery(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	if dsn := os.Getenv("GOPRESS_TEST_DSN"); dsn != "" {
		_ = dsn
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	return c
}

func TestGetPostsLatest(t *testing.T) {
	c := testBootQuery(t)
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostsPerPage: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected published posts")
	}
	for _, p := range posts {
		if p.Type != "post" || p.Status != "publish" {
			t.Fatalf("unexpected post: %+v", p)
		}
		if p.ID <= 0 || p.Title == "" {
			t.Fatalf("incomplete post: %+v", p)
		}
	}
}

func TestGetPostByID(t *testing.T) {
	c := testBootQuery(t)
	p, err := wp.GetPost(c, wp.QueryArgs{P: 1})
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || p.ID != 1 {
		t.Fatalf("expected post 1, got %+v", p)
	}
}

func TestGetPostByPageName(t *testing.T) {
	c := testBootQuery(t)
	// Sample page from fixture site; skip if absent.
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostType: "page", PostsPerPage: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Skip("no pages in fixture DB")
	}
	slug := posts[0].Name
	got, err := wp.GetPost(c, wp.QueryArgs{PageName: slug})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != slug || got.Type != "page" {
		t.Fatalf("expected page %q, got %+v", slug, got)
	}
}
