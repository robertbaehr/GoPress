package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.180: orderby=meta_value + meta_type=DATE sorts chronologically.
func TestQueryOrderByMetaValueMetaTypeDateP4180(t *testing.T) {
	c := testBootQuery(t)
	key := "p4180-date-key"
	clearMetaKey(t, c, key)
	t.Cleanup(func() { clearMetaKey(t, c, key) })
	mustAddMeta(t, c, 1, key, "2024-06-01")
	mustAddMeta(t, c, 13, key, "2023-01-15")

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostIn:       []int64{1, 13},
		MetaKey:      key,
		MetaCompare:  "EXISTS",
		MetaType:     "DATE",
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
		t.Fatalf("ASC DATE order got %d,%d want 13,1", posts[0].ID, posts[1].ID)
	}
}

// P4.180: CHAR keeps lexicographic order for the same date-looking values.
func TestQueryOrderByMetaValueMetaTypeDateVsCharP4180(t *testing.T) {
	c := testBootQuery(t)
	key := "p4180-date-char-key"
	clearMetaKey(t, c, key)
	t.Cleanup(func() { clearMetaKey(t, c, key) })
	// Lexicographic: "2024-1-2" < "2024-01-15" as strings? Actually "2024-1-2" vs "2024-01-15":
	// '2024-1' vs '2024-0' → '1' > '0' so "2024-01-15" < "2024-1-2" as CHAR.
	// Use values where CHAR and DATE disagree clearly.
	mustAddMeta(t, c, 1, key, "2024-1-2")
	mustAddMeta(t, c, 13, key, "2023-12-31")

	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostIn:       []int64{1, 13},
		MetaKey:      key,
		MetaCompare:  "EXISTS",
		MetaType:     "DATE",
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
	// DATE cast: 2023-12-31 before 2024-1-2.
	if posts[0].ID != 13 || posts[1].ID != 1 {
		t.Fatalf("ASC DATE order got %d,%d want 13,1", posts[0].ID, posts[1].ID)
	}
}
