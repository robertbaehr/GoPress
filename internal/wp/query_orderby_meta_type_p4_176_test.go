package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.176: orderby=meta_value + meta_type=NUMERIC casts like meta_value_num.
func TestQueryOrderByMetaValueMetaTypeNumericP4176(t *testing.T) {
	c := testBootQuery(t)
	key := "p4176-num-key"
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
		MetaType:     "NUMERIC",
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
		t.Fatalf("ASC meta_type=NUMERIC order got %d,%d want 13,1 (20 then 100)", posts[0].ID, posts[1].ID)
	}
}

// P4.176: CHAR (default) keeps string sort when meta_type omitted/CHAR.
func TestQueryOrderByMetaValueMetaTypeCharP4176(t *testing.T) {
	c := testBootQuery(t)
	key := "p4176-char-key"
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
	// Lexicographic: "100" < "20".
	if posts[0].ID != 1 || posts[1].ID != 13 {
		t.Fatalf("ASC CHAR order got %d,%d want 1,13", posts[0].ID, posts[1].ID)
	}
}
