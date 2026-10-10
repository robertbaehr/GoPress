package wp_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestHomeTemplateSlugs(t *testing.T) {
	if got := wp.HomeTemplateSlugs(true); !reflect.DeepEqual(got, []string{"front-page", "home", "index"}) {
		t.Fatalf("front: %v", got)
	}
	if got := wp.HomeTemplateSlugs(false); !reflect.DeepEqual(got, []string{"home", "index"}) {
		t.Fatalf("home: %v", got)
	}
}

func TestSearchTemplateSlugs(t *testing.T) {
	if got := wp.SearchTemplateSlugs(); !reflect.DeepEqual(got, []string{"search", "index"}) {
		t.Fatalf("search: %v", got)
	}
}

func TestDateTemplateSlugs(t *testing.T) {
	if got := wp.DateTemplateSlugs(); !reflect.DeepEqual(got, []string{"date", "archive", "index"}) {
		t.Fatalf("date: %v", got)
	}
}

func TestSingularTemplateSlugsPage(t *testing.T) {
	got := wp.SingularTemplateSlugs("page", "sample-page", 2)
	want := []string{"page-sample-page", "page-2", "page", "singular", "index"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSingularTemplateSlugsPost(t *testing.T) {
	got := wp.SingularTemplateSlugs("post", "hello-world", 1)
	want := []string{"single-post-hello-world", "single-post", "single", "singular", "index"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestResolveSingularHierarchyUsesPageThenSingular(t *testing.T) {
	c := bootWP(t)
	// Fixture theme has page.html; page-{slug}/singular may be absent.
	slugs := wp.SingularTemplateSlugs("page", "sample-page", 2)
	content, ok := wp.ResolveBlockTemplate(c, slugs...)
	if !ok {
		t.Fatal("expected page hierarchy resolve")
	}
	if !strings.Contains(content, "<!-- wp:") {
		t.Fatalf("expected block markup, got %q", content[:min(80, len(content))])
	}
}
