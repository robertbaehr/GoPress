package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.164: CategoryOperator AND emits stricter matching than default IN.
func TestGetPostsCategoryOperatorANDP4164(t *testing.T) {
	c := testBootQuery(t)

	inPosts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "post",
		PostStatus:   "publish",
		PostsPerPage: 100,
		CategoryIn:   []int64{1, 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	andPosts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:         "post",
		PostStatus:       "publish",
		PostsPerPage:     100,
		CategoryIn:       []int64{1, 2},
		CategoryOperator: "AND",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(andPosts) > len(inPosts) {
		t.Fatalf("AND=%d > IN=%d", len(andPosts), len(inPosts))
	}
}
