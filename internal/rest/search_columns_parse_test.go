package rest

import (
	"net/http/httptest"
	"testing"
)

func TestParsePostsSearchColumns(t *testing.T) {
	req := httptest.NewRequest("GET", "/x?search_columns=post_title,post_excerpt&search_columns[]=post_content", nil)
	cols, errMsg := parsePostsSearchColumns(req)
	if errMsg != "" {
		t.Fatalf("unexpected err %q", errMsg)
	}
	if len(cols) != 3 {
		t.Fatalf("got %v", cols)
	}
	want := map[string]bool{"post_title": true, "post_excerpt": true, "post_content": true}
	for _, c := range cols {
		if !want[c] {
			t.Fatalf("unexpected %q in %v", c, cols)
		}
	}
	cols, errMsg = parsePostsSearchColumns(nil)
	if cols != nil || errMsg != "" {
		t.Fatal("nil request")
	}
}

// P4.153: invalid enum token returns rest_not_in_enum message with index.
func TestParsePostsSearchColumnsInvalidP4153(t *testing.T) {
	req := httptest.NewRequest("GET", "/x?search_columns=bogus", nil)
	cols, errMsg := parsePostsSearchColumns(req)
	if cols != nil {
		t.Fatalf("cols=%v", cols)
	}
	want := "search_columns[0] is not one of post_title, post_content, and post_excerpt."
	if errMsg != want {
		t.Fatalf("errMsg=%q want %q", errMsg, want)
	}

	req = httptest.NewRequest("GET", "/x?search_columns=post_title,,bogus", nil)
	_, errMsg = parsePostsSearchColumns(req)
	want = "search_columns[1] is not one of post_title, post_content, and post_excerpt."
	if errMsg != want {
		t.Fatalf("mixed errMsg=%q want %q", errMsg, want)
	}

	req = httptest.NewRequest("GET", "/x?search_columns=", nil)
	cols, errMsg = parsePostsSearchColumns(req)
	if errMsg != "" || len(cols) != 0 {
		t.Fatalf("empty scalar: cols=%v err=%q", cols, errMsg)
	}
}
