package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.183: orderby=meta_value + meta_type=BINARY sorts by byte order (case-sensitive).
func TestQueryOrderByMetaValueMetaTypeBinaryP4183(t *testing.T) {
	c := testBootQuery(t)
	key := "p4183-binary-key"
	clearMetaKey(t, c, key)
	t.Cleanup(func() { clearMetaKey(t, c, key) })
	// ASCII: 'B' (66) before 'a' (97) under BINARY; case-insensitive CHAR may invert.
	mustAddMeta(t, c, 1, key, "apple")
	mustAddMeta(t, c, 13, key, "Banana")

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostIn:       []int64{1, 13},
		MetaKey:      key,
		MetaCompare:  "EXISTS",
		MetaType:     "BINARY",
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
		t.Fatalf("ASC BINARY order got %d,%d want 13,1 (Banana before apple)", posts[0].ID, posts[1].ID)
	}
}
