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
		ListArchives: func(archiveType string) []blocks.ArchiveItem {
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

func TestRenderArchivesDropdownPolish(t *testing.T) {
	var gotType string
	env := &blocks.Env{
		HomeURL: func() string { return "http://example.test" },
		ListArchives: func(archiveType string) []blocks.ArchiveItem {
			gotType = archiveType
			return []blocks.ArchiveItem{{Label: "2024", URL: "/2024/", Count: 5}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:archives {"displayAsDropdown":true,"showLabel":false,"type":"yearly","showPostCounts":true,"className":"is-style-default","style":{"elements":{"link":{"color":{"text":"var:preset|color|vivid-red"}}}}} /-->`)
	if gotType != "yearly" {
		t.Fatalf("archiveType=%q", gotType)
	}
	if !strings.Contains(out, `wp-block-archives-dropdown`) || !strings.Contains(out, `has-link-color`) {
		t.Fatalf("missing wrapper classes: %q", out)
	}
	if !strings.Contains(out, `screen-reader-text`) {
		t.Fatalf("expected hidden label: %q", out)
	}
	if !strings.Contains(out, `Select Year`) {
		t.Fatalf("expected yearly label: %q", out)
	}
	if !strings.Contains(out, `id="wp-block-archives-`) || !strings.Contains(out, `for="wp-block-archives-`) {
		t.Fatalf("expected unique dropdown id: %q", out)
	}
	if !strings.Contains(out, `location.href`) || !strings.Contains(out, `http://example.test`) {
		t.Fatalf("expected dropdown script: %q", out)
	}
	if !strings.Contains(out, `2024 (5)`) {
		t.Fatalf("missing option: %q", out)
	}
}

func TestRenderCalendarHiddenWhenNoPosts(t *testing.T) {
	env := &blocks.Env{
		HasPublishedPosts: func() bool { return false },
		IsLoggedIn:        func() bool { return true },
	}
	out := blocks.RenderWith(env, `<!-- wp:calendar /-->`)
	if !strings.Contains(out, `no published posts`) {
		t.Fatalf("expected empty-state notice: %q", out)
	}
	env.IsLoggedIn = func() bool { return false }
	if out := blocks.RenderWith(env, `<!-- wp:calendar /-->`); out != "" {
		t.Fatalf("guest should see empty calendar: %q", out)
	}
}

func TestRenderCalendarWrapperPolish(t *testing.T) {
	env := &blocks.Env{
		HomeURL:           func() string { return "http://example.test" },
		HasPublishedPosts: func() bool { return true },
	}
	out := blocks.RenderWith(env, `<!-- wp:calendar {"className":"is-style-default","anchor":"site-cal","textColor":"contrast","backgroundColor":"base","style":{"elements":{"link":{"color":{"text":"var:preset|color|vivid-red"}}}}} /-->`)
	if !strings.Contains(out, `wp-block-calendar`) || !strings.Contains(out, `is-style-default`) {
		t.Fatalf("missing wrapper: %q", out)
	}
	if !strings.Contains(out, `id="site-cal"`) {
		t.Fatalf("missing anchor: %q", out)
	}
	if !strings.Contains(out, `has-link-color`) || !strings.Contains(out, `has-contrast-color`) || !strings.Contains(out, `has-base-background-color`) {
		t.Fatalf("missing color classes: %q", out)
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
	if !strings.Contains(out, `has-dates`) {
		t.Fatalf("missing has-dates: %q", out)
	}
}

func TestRenderLatestPostsBlockDeepen(t *testing.T) {
	var got blocks.QueryLoopArgs
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			got = q
			return []blocks.QueryPost{{
				ID: 7, Title: "Grid Post", Link: "/grid/", Date: "2024-06-15 12:00:00",
				Excerpt: "One two three four five six", Content: "<p>Full body</p>",
			}}
		},
		AuthorName: func(postID int64) string {
			if postID != 7 {
				t.Fatalf("author post=%d", postID)
			}
			return "Admin"
		},
	}
	markup := `<!-- wp:latest-posts {"postsToShow":3,"displayPostDate":true,"displayAuthor":true,"displayPostContent":true,"displayPostContentRadio":"excerpt","excerptLength":3,"order":"asc","orderBy":"title","selectedAuthor":2,"categories":[{"id":1}],"postLayout":"grid","columns":3,"className":"lp-extra","style":{"elements":{"link":{"color":{"text":"red"}}}}} /-->`
	out := blocks.RenderWith(env, markup)
	if got.PerPage != 3 || got.Order != "asc" || got.OrderBy != "title" || got.Author != 2 {
		t.Fatalf("query args=%+v", got)
	}
	if len(got.CategoryIn) != 1 || got.CategoryIn[0] != 1 {
		t.Fatalf("categories=%v", got.CategoryIn)
	}
	if got.Sticky != "ignore" {
		t.Fatalf("sticky=%q", got.Sticky)
	}
	for _, want := range []string{
		`is-grid`, `columns-3`, `has-dates`, `has-author`,
		`wp-block-latest-posts__post-author`, `by Admin`,
		`wp-block-latest-posts__post-excerpt`, `One two three`,
		`lp-extra`, `has-link-color`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderLatestCommentsBlock(t *testing.T) {
	env := &blocks.Env{
		ListLatestComments: func(limit int) []blocks.LatestCommentItem {
			return []blocks.LatestCommentItem{{
				Author: "Alice", AuthorEmail: "a@example.com", Content: "Nice post",
				PostTitle: "Hello", PostURL: "/hello/#comment-1", Date: "2024-06-15",
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
	if !strings.Contains(out, `has-avatars`) || !strings.Contains(out, `wp-block-latest-comments__comment-avatar`) {
		t.Fatalf("missing avatar polish: %q", out)
	}
}

func TestRenderLatestCommentsBlockNoAvatar(t *testing.T) {
	env := &blocks.Env{
		ListLatestComments: func(limit int) []blocks.LatestCommentItem {
			return []blocks.LatestCommentItem{{Author: "Bob", Content: "Hi", PostTitle: "P", PostURL: "/p/"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:latest-comments {"displayAvatar":false,"displayDate":false,"displayContent":"none","className":"lc-extra"} /-->`)
	if strings.Contains(out, `has-avatars`) || strings.Contains(out, `comment-avatar`) {
		t.Fatalf("avatar should be off: %q", out)
	}
	if !strings.Contains(out, `lc-extra`) {
		t.Fatalf("missing className: %q", out)
	}
}

func TestRenderTagCloudBlock(t *testing.T) {
	env := &blocks.Env{
		ListTagCloud: func(taxonomy string, number int) []blocks.TagCloudItem {
			if taxonomy != "post_tag" {
				t.Fatalf("taxonomy=%q", taxonomy)
			}
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

func TestRenderTagCloudNumberAndTaxonomy(t *testing.T) {
	var gotTax string
	var gotNum int
	env := &blocks.Env{
		ListTagCloud: func(taxonomy string, number int) []blocks.TagCloudItem {
			gotTax, gotNum = taxonomy, number
			return []blocks.TagCloudItem{
				{Name: "a", URL: "/category/a/", Count: 3},
				{Name: "b", URL: "/category/b/", Count: 2},
				{Name: "c", URL: "/category/c/", Count: 1},
			}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:tag-cloud {"taxonomy":"category","numberOfTags":2,"showTagCounts":true} /-->`)
	if gotTax != "category" || gotNum != 2 {
		t.Fatalf("ListTagCloud args taxonomy=%q number=%d", gotTax, gotNum)
	}
	if strings.Contains(out, "category/c") || strings.Contains(out, ">c ") {
		t.Fatalf("expected numberOfTags=2 to truncate: %q", out)
	}
	if !strings.Contains(out, `>a (3)<`) || !strings.Contains(out, `>b (2)<`) {
		t.Fatalf("expected first two terms: %q", out)
	}
}
