package rest

import (
	"net/http/httptest"
	"testing"
)

func TestParsePostsSearchColumns(t *testing.T) {
	req := httptest.NewRequest("GET", "/x?search_columns=post_title,post_excerpt&search_columns[]=post_content&search_columns=bogus", nil)
	cols := parsePostsSearchColumns(req)
	if len(cols) != 3 {
		t.Fatalf("got %v", cols)
	}
	want := map[string]bool{"post_title": true, "post_excerpt": true, "post_content": true}
	for _, c := range cols {
		if !want[c] {
			t.Fatalf("unexpected %q in %v", c, cols)
		}
	}
	if parsePostsSearchColumns(nil) != nil {
		t.Fatal("nil request")
	}
}
