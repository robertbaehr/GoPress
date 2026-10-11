package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.182: orderby=meta_value + meta_type=SIGNED sorts numerically (signed).
func TestQueryOrderByMetaValueMetaTypeSignedP4182(t *testing.T) {
	c := testBootQuery(t)
	key := "p4182-signed-key"
	clearMetaKey(t, c, key)
	t.Cleanup(func() { clearMetaKey(t, c, key) })
	mustAddMeta(t, c, 1, key, "100")
	mustAddMeta(t, c, 13, key, "20")

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostIn:       []int64{1, 13},
		MetaKey:      key,
		MetaCompare:  "EXISTS",
		MetaType:     "SIGNED",
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
		t.Fatalf("ASC SIGNED order got %d,%d want 13,1", posts[0].ID, posts[1].ID)
	}
}
