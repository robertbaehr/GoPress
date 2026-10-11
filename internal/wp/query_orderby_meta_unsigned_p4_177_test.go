package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.177: orderby=meta_value + meta_type=UNSIGNED sorts numerically (unsigned).
func TestQueryOrderByMetaValueMetaTypeUnsignedP4177(t *testing.T) {
	c := testBootQuery(t)
	key := "p4177-unsigned-key"
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
		MetaType:     "UNSIGNED",
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
		t.Fatalf("ASC UNSIGNED order got %d,%d want 13,1", posts[0].ID, posts[1].ID)
	}
}

// P4.177: meta_value_num keeps SIGNED cast even when MetaType is CHAR.
func TestQueryOrderByMetaValueNumIgnoresMetaTypeP4177(t *testing.T) {
	c := testBootQuery(t)
	key := "p4177-num-ignore-type"
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
		OrderBy:      "meta_value_num",
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
		t.Fatalf("meta_value_num should stay numeric got %d,%d", posts[0].ID, posts[1].ID)
	}
}
