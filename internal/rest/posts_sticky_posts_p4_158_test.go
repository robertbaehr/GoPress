package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.158: sticky=true with empty sticky_posts → zero results (post__in=[0]).
func TestRESTStickyTrueEmptyStickyPostsP4158(t *testing.T) {
	c := bootOrSkip(t)
	const fixtureSticky = "a:1:{i:0;i:13;}"
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("sticky_posts", fixtureSticky, "yes")
		_ = c.DB.Close()
	})
	_, _ = c.Options.UpdateOption("sticky_posts", "a:0:{}", "yes")

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?sticky=true&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(posts) != 0 {
		t.Fatalf("empty sticky_posts + sticky=true want 0 posts, got %d", len(posts))
	}
}

// P4.158: is_sticky filter flips REST sticky field.
func TestRESTStickyFieldIsStickyFilterP4158(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	// Fixture sticky_posts includes 13; force filter override for that ID.
	c.Hooks.AddFilter("is_sticky", func(value any, args ...any) any {
		if len(args) > 0 {
			if id, ok := args[0].(int64); ok && id == int64(13) {
				return false
			}
		}
		return value
	}, 10, 2)
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/13", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("json: %v", err)
	}
	if post.Sticky {
		t.Fatalf("is_sticky filter should clear sticky on post 13")
	}
}
