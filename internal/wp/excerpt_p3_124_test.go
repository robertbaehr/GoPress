package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestHasExcerpt(t *testing.T) {
	if wp.HasExcerpt(nil) || wp.HasExcerpt(&wp.Post{}) {
		t.Fatal("empty should be false")
	}
	if !wp.HasExcerpt(&wp.Post{Excerpt: " Hi "}) {
		t.Fatal("expected true")
	}
}

func TestGetTheExcerptCustom(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	post := &wp.Post{ID: 1, Excerpt: "Custom tease", Content: "Long body"}
	if got := wp.GetTheExcerpt(c, post); got != "Custom tease" {
		t.Fatalf("custom: %q", got)
	}
}

func TestGetTheExcerptTrimFromContent(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	words := strings.Repeat("word ", 60)
	post := &wp.Post{ID: 1, Content: "<p>" + words + "</p>"}
	got := wp.GetTheExcerpt(c, post)
	if strings.Contains(got, "<p>") {
		t.Fatalf("should strip tags: %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis trim: %q", got)
	}
	if len(strings.Fields(got)) != 55 {
		// 55 words + ellipsis attached to last — Fields still 55
		t.Fatalf("word count=%d got=%q", len(strings.Fields(strings.TrimSuffix(got, "…"))), got)
	}
}

func TestGetTheExcerptPasswordProtected(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Req = req
	post := &wp.Post{ID: 9, Password: "secret", Excerpt: "Hidden"}
	got := wp.GetTheExcerpt(c, post)
	if got != "There is no excerpt because this is a protected post." {
		t.Fatalf("protected: %q", got)
	}
}

func TestGetTheExcerptFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("get_the_excerpt", func(value any, _ ...any) any {
		return "filtered-excerpt"
	}, 10, 2)
	got := wp.GetTheExcerpt(c, &wp.Post{Excerpt: "raw"})
	if got != "filtered-excerpt" {
		t.Fatalf("filter: %q", got)
	}
}

func TestTheExcerptAutopAndFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("the_excerpt", func(value any, _ ...any) any {
		s, _ := value.(string)
		return s + "<!--x-->"
	}, 10, 1)
	got := wp.TheExcerpt(c, &wp.Post{Excerpt: "Plain line"})
	if !strings.Contains(got, "<p>") || !strings.Contains(got, "<!--x-->") {
		t.Fatalf("the_excerpt: %q", got)
	}
}
