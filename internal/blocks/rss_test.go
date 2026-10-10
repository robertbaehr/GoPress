package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderRSSLocalFeed(t *testing.T) {
	env := &blocks.Env{
		HomeURL: func() string { return "http://127.0.0.1:8082/" },
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			if q.PerPage != 2 {
				t.Fatalf("itemsToShow → PerPage=%d", q.PerPage)
			}
			return []blocks.QueryPost{
				{ID: 1, Title: "Alpha", Link: "/alpha/", Date: "2024-02-01T12:00:00"},
				{ID: 2, Title: "Beta", Link: "/beta/", Date: "2024-02-02T12:00:00"},
			}
		},
		AuthorName: func(postID int64) string {
			if postID == 1 {
				return "Admin"
			}
			return "Editor"
		},
	}
	raw := `<!-- wp:rss {"feedURL":"http://127.0.0.1:8082/feed/","itemsToShow":2,"displayDate":true,"displayAuthor":true} /-->`
	out := blocks.RenderWith(env, raw)
	if !strings.Contains(out, `class="wp-block-rss`) {
		t.Fatalf("expected wp-block-rss, got %q", out)
	}
	if !strings.Contains(out, `wp-block-rss__item-title`) || !strings.Contains(out, "Alpha") {
		t.Fatalf("expected item titles, got %q", out)
	}
	if !strings.Contains(out, `has-dates`) || !strings.Contains(out, `wp-block-rss__item-publish-date`) {
		t.Fatalf("expected dates, got %q", out)
	}
	if !strings.Contains(out, `has-authors`) || !strings.Contains(out, "by Admin") {
		t.Fatalf("expected authors, got %q", out)
	}
}

func TestRenderRSSRelativeLocalFeed(t *testing.T) {
	env := &blocks.Env{
		HomeURL: func() string { return "http://example.test/" },
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 3, Title: "Local", Link: "/local/"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:rss {"feedURL":"/feed/rss2/"} /-->`)
	if !strings.Contains(out, "Local") || !strings.Contains(out, "wp-block-rss__item") {
		t.Fatalf("relative local feed: %q", out)
	}
}

func TestRenderRSSRemoteBlocked(t *testing.T) {
	env := &blocks.Env{
		HomeURL: func() string { return "http://127.0.0.1:8082/" },
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			t.Fatal("ListPosts should not run for remote feeds")
			return nil
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:rss {"feedURL":"https://wordpress.org/news/feed/"} /-->`)
	if !strings.Contains(out, "RSS Error:") || !strings.Contains(out, "Remote feeds are blocked.") {
		t.Fatalf("expected remote blocked notice, got %q", out)
	}
}

func TestRenderRSSSiteRootLoopGuard(t *testing.T) {
	env := &blocks.Env{
		HomeURL: func() string { return "http://127.0.0.1:8082/" },
	}
	out := blocks.RenderWith(env, `<!-- wp:rss {"feedURL":"http://127.0.0.1:8082"} /-->`)
	if !strings.Contains(out, "homepage is not supported") {
		t.Fatalf("expected loop guard, got %q", out)
	}
}

func TestRenderRSSEmptyFeedURL(t *testing.T) {
	out := blocks.Render(`<!-- wp:rss /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("empty feedURL should render empty, got %q", out)
	}
}
