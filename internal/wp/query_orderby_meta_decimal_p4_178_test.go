package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.178: orderby=meta_value + meta_type=DECIMAL sorts numerically.
func TestQueryOrderByMetaValueMetaTypeDecimalP4178(t *testing.T) {
	c := testBootQuery(t)
	key := "p4178-decimal-key"
	clearMetaKey(t, c, key)
	t.Cleanup(func() { clearMetaKey(t, c, key) })
	mustAddMeta(t, c, 1, key, "10.50")
	mustAddMeta(t, c, 13, key, "2.75")

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostIn:       []int64{1, 13},
		MetaKey:      key,
		MetaCompare:  "EXISTS",
		MetaType:     "DECIMAL",
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
		t.Fatalf("ASC DECIMAL order got %d,%d want 13,1 (2.75 then 10.50)", posts[0].ID, posts[1].ID)
	}
}

// P4.178: CHAR keeps lexicographic order for the same decimal-looking values.
func TestQueryOrderByMetaValueMetaTypeCharVsDecimalP4178(t *testing.T) {
	c := testBootQuery(t)
	key := "p4178-char-vs-decimal"
	clearMetaKey(t, c, key)
	t.Cleanup(func() { clearMetaKey(t, c, key) })
	mustAddMeta(t, c, 1, key, "10.50")
	mustAddMeta(t, c, 13, key, "2.75")

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostIn:       []int64{1, 13},
		MetaKey:      key,
		MetaCompare:  "EXISTS",
		MetaType:     "CHAR",
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
	// Lexicographic: "10.50" < "2.75"
	if posts[0].ID != 1 || posts[1].ID != 13 {
		t.Fatalf("ASC CHAR order got %d,%d want 1,13", posts[0].ID, posts[1].ID)
	}
}
