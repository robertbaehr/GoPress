package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.179: orderby=meta_value + meta_type=DATETIME sorts chronologically.
func TestQueryOrderByMetaValueMetaTypeDatetimeP4179(t *testing.T) {
	c := testBootQuery(t)
	key := "p4179-datetime-key"
	clearMetaKey(t, c, key)
	t.Cleanup(func() { clearMetaKey(t, c, key) })
	mustAddMeta(t, c, 1, key, "2024-06-01 10:00:00")
	mustAddMeta(t, c, 13, key, "2023-01-15 08:00:00")

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostIn:       []int64{1, 13},
		MetaKey:      key,
		MetaCompare:  "EXISTS",
		MetaType:     "DATETIME",
		OrderBy:      "meta_value",
		Order:        "ASC",
		PostsPerPage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("len=%d", len(posts))
	}
	if posts[0].ID != 13 || posts[1].ID != 1 {
		t.Fatalf("ASC DATETIME order got %d,%d want 13,1", posts[0].ID, posts[1].ID)
	}
}
