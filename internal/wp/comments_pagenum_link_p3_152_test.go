package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.152: pretty comments pagenum link + #comments + filter.
func TestGetCommentsPagenumLinkPrettyP3152(t *testing.T) {
	c := testBootQuery(t)
	_, _ = c.Options.UpdateOption("default_comments_page", "oldest", "yes")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("default_comments_page", "newest", "yes") })

	got := wp.GetCommentsPagenumLink(c, "http://example.test/hello/", 2, 5, true)
	want := "http://example.test/hello/comment-page-2/#comments"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	c.Hooks.AddFilter("get_comments_pagenum_link", func(value any, _ ...any) any {
		return "https://filtered.example/#comments"
	}, 10, 1)
	if got := wp.GetCommentsPagenumLink(c, "http://example.test/hello/", 2, 5, true); got != "https://filtered.example/#comments" {
		t.Fatalf("filter=%q", got)
	}
}

// P3.152: newest default omits page when pagenum == max_page.
func TestGetCommentsPagenumLinkNewestOmitP3152(t *testing.T) {
	c := testBootQuery(t)
	_, _ = c.Options.UpdateOption("default_comments_page", "newest", "yes")

	got := wp.GetCommentsPagenumLink(c, "http://example.test/hello/", 3, 3, true)
	if got != "http://example.test/hello/#comments" {
		t.Fatalf("newest last page=%q", got)
	}
	got = wp.GetCommentsPagenumLink(c, "http://example.test/hello/", 1, 3, false)
	if !strings.Contains(got, "cpage=1") || !strings.HasSuffix(got, "#comments") {
		t.Fatalf("newest earlier page query=%q", got)
	}
}

func TestGetCommentsPagenumLinkEmptyP3152(t *testing.T) {
	if got := wp.GetCommentsPagenumLink(nil, "", 1, 1, true); got != "" {
		t.Fatalf("empty=%q", got)
	}
}
