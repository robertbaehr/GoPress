package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.166: MetaKey/MetaValue EXISTS filter via QueryArgs.
func TestMetaQueryEqualsFilterP4166(t *testing.T) {
	c := testBootQuery(t)

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostsPerPage: 5,
		MetaKey:      "footnotes",
		MetaValue:    "unlikely-meta-value-p4166",
		MetaCompare:  "=",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 0 {
		t.Fatalf("expected empty for missing meta, got %d", len(posts))
	}

	// EXISTS without value should not error.
	_, err = wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostsPerPage: 5,
		MetaKey:      "footnotes",
		MetaCompare:  "EXISTS",
	})
	if err != nil {
		t.Fatal(err)
	}
}
