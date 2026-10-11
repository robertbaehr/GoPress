package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func commentPagingCtx(t *testing.T, opts map[string]string) *wp.Context {
	t.Helper()
	cache := wp.NewObjectCache()
	for k, v := range opts {
		_ = cache.Set(k, v, "options")
	}
	return &wp.Context{Options: wp.NewOptions(nil, cache), Hooks: wp.NewHooks()}
}

func TestCommentPagingDisabledByDefault(t *testing.T) {
	c := commentPagingCtx(t, map[string]string{
		"page_comments":         "0",
		"comments_per_page":     "2",
		"default_comments_page": "newest",
	})
	p := wp.CommentPagingForPost(c, 5)
	if p.Enabled {
		t.Fatalf("expected disabled: %+v", p)
	}
	comments := []int{1, 2, 3, 4, 5}
	got := wp.SliceCommentPage(comments, p)
	if len(got) != 5 {
		t.Fatalf("disabled should keep all: %v", got)
	}
}

func TestCommentPagingNewestDefaultAndCpage(t *testing.T) {
	c := commentPagingCtx(t, map[string]string{
		"page_comments":         "1",
		"comments_per_page":     "2",
		"default_comments_page": "newest",
	})
	c.Req = httptest.NewRequest(http.MethodGet, "/hello/", nil)
	p := wp.CommentPagingForPost(c, 5)
	if !p.Enabled || p.Total != 3 || p.Page != 3 {
		t.Fatalf("newest default want page 3/3: %+v", p)
	}
	comments := []string{"a", "b", "c", "d", "e"}
	got := wp.SliceCommentPage(comments, p)
	if strings.Join(got, ",") != "e" {
		t.Fatalf("last page slice=%v", got)
	}

	c.Req = httptest.NewRequest(http.MethodGet, "/hello/?cpage=1", nil)
	p = wp.CommentPagingForPost(c, 5)
	if p.Page != 1 {
		t.Fatalf("cpage=1: %+v", p)
	}
	got = wp.SliceCommentPage(comments, p)
	if strings.Join(got, ",") != "a,b" {
		t.Fatalf("page1=%v", got)
	}
}

func TestCommentPagingOldestDefault(t *testing.T) {
	c := commentPagingCtx(t, map[string]string{
		"page_comments":         "1",
		"comments_per_page":     "2",
		"default_comments_page": "oldest",
	})
	c.Req = httptest.NewRequest(http.MethodGet, "/hello/", nil)
	p := wp.CommentPagingForPost(c, 5)
	if p.Page != 1 || p.Total != 3 {
		t.Fatalf("oldest default: %+v", p)
	}
}

func TestCommentsPagenumLinkAndNav(t *testing.T) {
	link := wp.CommentsPagenumLink("http://example.test/hello/", 2)
	if link != "http://example.test/hello/?cpage=2" {
		t.Fatalf("link=%q", link)
	}
	link = wp.CommentsPagenumLink("http://example.test/?p=13", 2)
	if !strings.Contains(link, "cpage=2") || !strings.Contains(link, "p=13") {
		t.Fatalf("query merge=%q", link)
	}
	nav := wp.CommentPaginationHTML("http://example.test/hello/", wp.CommentPaging{
		Enabled: true, PerPage: 2, Page: 2, Total: 3,
	})
	if !strings.Contains(nav, "comment-navigation") || !strings.Contains(nav, "cpage=1") || !strings.Contains(nav, "cpage=3") {
		t.Fatalf("nav=%q", nav)
	}
	if !strings.Contains(nav, `aria-current="page"`) || !strings.Contains(nav, "#comments") {
		t.Fatalf("nav current/fragment=%q", nav)
	}
}
