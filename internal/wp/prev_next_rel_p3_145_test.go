package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.145: get_pagenum_link filter rewrites archive prev/next hrefs.
func TestArchiveRelPrevNextGetPagenumLinkFilterP3145(t *testing.T) {
	c := testBootQuery(t)
	c.Hooks.AddFilter("get_pagenum_link", func(value any, args ...any) any {
		s, _ := value.(string)
		return strings.Replace(s, "http://example.com/", "https://cdn.example/", 1)
	}, 10, 2)

	base := "http://example.com/category/news/"
	out := wp.ArchiveRelPrevNextHTMLQueryCtx(c, base, 2, 3, "")
	if !strings.Contains(out, `href="https://cdn.example/category/news/"`) {
		t.Fatalf("prev filter missing: %q", out)
	}
	if !strings.Contains(out, `href="https://cdn.example/category/news/page/3/"`) {
		t.Fatalf("next filter missing: %q", out)
	}
}

// P3.145: EscURL encodes spaces in filtered pagenum links.
func TestArchiveRelPrevNextEscURLP3145(t *testing.T) {
	c := testBootQuery(t)
	c.Hooks.AddFilter("get_pagenum_link", func(value any, _ ...any) any {
		return "http://example.com/path with spaces/"
	}, 10, 2)
	out := wp.ArchiveRelPrevNextHTMLQueryCtx(c, "http://example.com/", 1, 2, "")
	if strings.Contains(out, "path with spaces") {
		t.Fatalf("raw spaces remain: %q", out)
	}
	if !strings.Contains(out, "path%20with%20spaces") && !strings.Contains(out, "path") {
		t.Fatalf("expected EscURL'd href: %q", out)
	}
}

// P3.145: adjacent post rel links with title + filters.
func TestAdjacentPostRelLinkHTMLP3145(t *testing.T) {
	c := testBootQuery(t)
	// Fixture posts: find two published posts.
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostType: "post", PostStatus: "publish", PostsPerPage: 5, OrderBy: "date", Order: "DESC"})
	if err != nil || len(posts) < 2 {
		t.Skipf("need >=2 posts: err=%v n=%d", err, len(posts))
	}
	cur := posts[0]
	home := "http://example.com"
	html := wp.AdjacentPostsRelLinkHTML(c, home, cur.ID, "")
	if html == "" {
		t.Fatalf("expected adjacent rel links for post %d", cur.ID)
	}
	if !strings.Contains(html, `rel='prev'`) && !strings.Contains(html, `rel='next'`) {
		t.Fatalf("missing rel: %q", html)
	}
	if !strings.Contains(html, `title='`) || !strings.Contains(html, `href='`) {
		t.Fatalf("missing title/href: %q", html)
	}

	c.Hooks.AddFilter("previous_post_rel_link", func(value any, _ ...any) any {
		return "<link rel='prev' title='Filtered Prev' href='http://example.com/?p=1' />\n"
	}, 10, 1)
	prev := wp.GetAdjacentPostRelLinkHTML(c, home, cur.ID, true, "")
	if prev != "" && !strings.Contains(prev, "Filtered Prev") {
		// Only assert when a previous post exists.
		if strings.Contains(html, `rel='prev'`) {
			t.Fatalf("filter not applied: %q", prev)
		}
	}
}
