package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.165: date_query inclusive filters on post_date / post_modified.
func TestDateQueryFiltersInclusiveP4165(t *testing.T) {
	c := testBootQuery(t)

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostsPerPage: 100,
		DateAfter:    "2000-01-01 00:00:00",
		DateBefore:   "2099-12-31 23:59:59",
	})
	if err != nil {
		t.Fatalf("GetPosts: %v", err)
	}
	if len(posts) == 0 {
		t.Fatal("expected published posts in wide date_query window")
	}

	// Exact-boundary inclusive: a known fixture post_date should be retained when after=that stamp.
	stamp := strings.TrimSpace(posts[0].Date)
	if stamp == "" {
		t.Fatal("empty post date")
	}
	boundary, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostsPerPage: 100,
		DateAfter:    stamp,
		PostIn:       []int64{posts[0].ID},
	})
	if err != nil {
		t.Fatalf("boundary GetPosts: %v", err)
	}
	found := false
	for _, p := range boundary {
		if p.ID == posts[0].ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("inclusive after=%q should retain post %d", stamp, posts[0].ID)
	}
}
