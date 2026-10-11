package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.175: orderby=meta_value sorts by meta_key string values.
func TestQueryOrderByMetaValueP4175(t *testing.T) {
	c := testBootQuery(t)
	key := "p4175-sort-key"
	clearMetaKey(t, c, key)
	t.Cleanup(func() { clearMetaKey(t, c, key) })
	mustAddMeta(t, c, 1, key, "zebra")
	mustAddMeta(t, c, 13, key, "apple")

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostIn:       []int64{1, 13},
		MetaKey:      key,
		MetaCompare:  "EXISTS",
		OrderBy:      "meta_value",
		Order:        "ASC",
		PostsPerPage: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("len=%d posts=%+v", len(posts), posts)
	}
	if posts[0].ID != 13 || posts[1].ID != 1 {
		t.Fatalf("ASC meta_value order got %d,%d want 13,1", posts[0].ID, posts[1].ID)
	}
}

// P4.175: orderby=meta_value_num sorts numerically.
func TestQueryOrderByMetaValueNumP4175(t *testing.T) {
	c := testBootQuery(t)
	key := "p4175-num-key"
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
		t.Fatalf("ASC meta_value_num order got %d,%d want 13,1 (20 then 100)", posts[0].ID, posts[1].ID)
	}
}

// P4.175: meta_value orderby without MetaKey falls back to date.
func TestQueryOrderByMetaValueMissingKeyP4175(t *testing.T) {
	c := testBootQuery(t)
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:   "post",
		PostStatus: "publish",
		OrderBy:    "meta_value",
		Order:      "DESC",
		PostsPerPage: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected posts with date fallback")
	}
}

func clearMetaKey(t *testing.T, c *wp.Context, key string) {
	t.Helper()
	q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.PostMeta()+" WHERE meta_key = %s", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}

func mustAddMeta(t *testing.T, c *wp.Context, postID int64, key, val string) {
	t.Helper()
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.PostMeta()+" (post_id, meta_key, meta_value) VALUES (%d, %s, %s)",
		postID, key, val,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}
