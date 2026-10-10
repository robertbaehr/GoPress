package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderSearchBlock(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test" },
		SearchQuery: "Hello",
	}
	out := blocks.RenderWith(env, `<!-- wp:search {"label":"Search","showLabel":false,"placeholder":"Type here...","buttonText":"Search"} /-->`)
	if !strings.Contains(out, `role="search"`) {
		t.Fatalf("missing form role: %q", out)
	}
	if !strings.Contains(out, `class="wp-block-search__button-outside`) {
		t.Fatalf("missing search classes: %q", out)
	}
	if !strings.Contains(out, `placeholder="Type here..."`) {
		t.Fatalf("missing placeholder: %q", out)
	}
	if !strings.Contains(out, `value="Hello"`) {
		t.Fatalf("missing value: %q", out)
	}
	if !strings.Contains(out, `name="s"`) || !strings.Contains(out, `type="submit"`) {
		t.Fatalf("missing input/button: %q", out)
	}
	if !strings.Contains(out, `screen-reader-text`) {
		t.Fatalf("label should be screen-reader-text: %q", out)
	}
}

func TestRenderCalendarBlock(t *testing.T) {
	env := &blocks.Env{HomeURL: func() string { return "http://example.test" }}
	out := blocks.RenderWith(env, `<!-- wp:calendar /-->`)
	if !strings.Contains(out, `class="wp-block-calendar"`) {
		t.Fatalf("missing calendar: %q", out)
	}
	if !strings.Contains(out, `<table class="wp-calendar-table"`) {
		t.Fatalf("missing table: %q", out)
	}
	if !strings.Contains(out, `<caption>`) {
		t.Fatalf("missing caption: %q", out)
	}
}

func TestRenderArchivesBlock(t *testing.T) {
	env := &blocks.Env{
		ListArchives: func() []blocks.ArchiveItem {
			return []blocks.ArchiveItem{{Label: "June 2024", URL: "/2024/06/", Count: 3}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:archives {"showPostCounts":true} /-->`)
	if !strings.Contains(out, `class="wp-block-archives`) {
		t.Fatalf("missing archives: %q", out)
	}
	if !strings.Contains(out, `June 2024`) || !strings.Contains(out, `(3)`) {
		t.Fatalf("missing item: %q", out)
	}
}

func TestRenderLatestPostsBlock(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			if q.PerPage != 2 {
				t.Fatalf("perPage=%d", q.PerPage)
			}
			return []blocks.QueryPost{{Title: "Hello world!", Link: "/hello/", Date: "2024-06-15 12:00:00"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:latest-posts {"postsToShow":2,"displayPostDate":true} /-->`)
	if !strings.Contains(out, `wp-block-latest-posts__list`) {
		t.Fatalf("missing list: %q", out)
	}
	if !strings.Contains(out, `Hello world!`) {
		t.Fatalf("missing title: %q", out)
	}
	if !strings.Contains(out, `wp-block-latest-posts__post-date`) {
		t.Fatalf("missing date: %q", out)
	}
}

func TestRenderLatestCommentsBlock(t *testing.T) {
	env := &blocks.Env{
		ListLatestComments: func(limit int) []blocks.LatestCommentItem {
			return []blocks.LatestCommentItem{{
				Author: "Alice", Content: "Nice post", PostTitle: "Hello", PostURL: "/hello/#comment-1", Date: "2024-06-15",
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:latest-comments {"commentsToShow":5} /-->`)
	if !strings.Contains(out, `wp-block-latest-comments`) {
		t.Fatalf("missing list: %q", out)
	}
	if !strings.Contains(out, `Alice`) || !strings.Contains(out, `Hello`) {
		t.Fatalf("missing meta: %q", out)
	}
}

func TestRenderTagCloudBlock(t *testing.T) {
	env := &blocks.Env{
		ListTagCloud: func() []blocks.TagCloudItem {
			return []blocks.TagCloudItem{{Name: "fixture-tag", URL: "/tag/fixture-tag/", Count: 2}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:tag-cloud {"showTagCounts":true} /-->`)
	if !strings.Contains(out, `wp-block-tag-cloud`) {
		t.Fatalf("missing cloud: %q", out)
	}
	if !strings.Contains(out, `fixture-tag`) || !strings.Contains(out, `(2)`) {
		t.Fatalf("missing tag: %q", out)
	}
}
