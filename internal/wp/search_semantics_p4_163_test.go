package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.163: multi-word AND search, stopwords dropped, exclusion prefix, quoted phrase.
func TestGetPostsSearchMultiWordANDP4163(t *testing.T) {
	c := testBootQuery(t)
	// Both "Fixture" and "Post" must match (AND) — Fixture Post 01 etc.
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostsPerPage: 20, Search: "Fixture Post"})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected Fixture Post hits")
	}
	for _, p := range posts {
		hay := strings.ToLower(p.Title + " " + p.Content + " " + p.Excerpt)
		if !strings.Contains(hay, "fixture") || !strings.Contains(hay, "post") {
			t.Fatalf("AND semantics failed for %q", p.Title)
		}
	}
}

func TestGetPostsSearchStopwordsSentenceFallbackP4163(t *testing.T) {
	c := testBootQuery(t)
	// "about the" are stopwords only → fall back to whole sentence LIKE.
	// Searching "the" alone (stopword) falls back to sentence "the".
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostsPerPage: 5, Search: "the"})
	if err != nil {
		t.Fatal(err)
	}
	// May or may not hit depending on content; just ensure no error and SQL ran.
	_ = posts

	// "Hello world" keeps both terms (neither is a stopword).
	posts, err = wp.GetPosts(c, wp.QueryArgs{PostsPerPage: 10, Search: "Hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected Hello world hits")
	}
}

func TestGetPostsSearchExclusionPrefixP4163(t *testing.T) {
	c := testBootQuery(t)
	// "Fixture -accordion" → must contain Fixture AND must NOT contain accordion.
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostsPerPage: 50, Search: "Fixture -accordion"})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected Fixture hits excluding accordion")
	}
	for _, p := range posts {
		hay := strings.ToLower(p.Title + " " + p.Content + " " + p.Excerpt)
		if !strings.Contains(hay, "fixture") {
			t.Fatalf("missing fixture in %q", p.Title)
		}
		if strings.Contains(hay, "accordion") {
			t.Fatalf("exclusion failed for %q", p.Title)
		}
	}
}

func TestGetPostsSearchQuotedPhraseP4163(t *testing.T) {
	c := testBootQuery(t)
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostsPerPage: 10, Search: `"Hello world"`})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected quoted phrase hit for Hello world")
	}
	found := false
	for _, p := range posts {
		if strings.Contains(strings.ToLower(p.Title), "hello world") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected title containing Hello world, got %+v", titlesOf(posts))
	}
}

func TestGetPostsSearchColumnsFilterP4163(t *testing.T) {
	c := testBootQuery(t)
	c.Hooks.AddFilter("post_search_columns", func(value any, _ ...any) any {
		return []string{"post_title"}
	}, 10, 2)
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostsPerPage:  10,
		Search:        "Hello",
		SearchColumns: []string{"post_content"}, // filter overrides to title-only
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected title-only Hello hits")
	}
	for _, p := range posts {
		if !strings.Contains(strings.ToLower(p.Title), "hello") {
			t.Fatalf("filter should restrict to title; got %q", p.Title)
		}
	}
}

func titlesOf(posts []wp.Post) []string {
	out := make([]string, len(posts))
	for i, p := range posts {
		out[i] = p.Title
	}
	return out
}
