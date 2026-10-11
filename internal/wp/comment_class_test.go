package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBuildCommentClassParity(t *testing.T) {
	classes, nextAlt, nextThread := wp.BuildCommentClass("comment", 0, "", 0, 1, 0, 0)
	joined := strings.Join(classes, " ")
	for _, want := range []string{"comment", "even", "thread-even", "depth-1"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if nextAlt != 1 || nextThread != 1 {
		t.Fatalf("nextAlt=%d nextThread=%d", nextAlt, nextThread)
	}
	classes, nextAlt2, nextThread2 := wp.BuildCommentClass("comment", 1, "admin", 1, 1, nextAlt, nextThread)
	joined = strings.Join(classes, " ")
	for _, want := range []string{"byuser", "comment-author-admin", "bypostauthor", "odd", "alt", "thread-odd", "thread-alt"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if nextAlt2 != 2 || nextThread2 != 2 {
		t.Fatalf("nextAlt2=%d nextThread2=%d", nextAlt2, nextThread2)
	}
}

func TestGetCommentClassFilter(t *testing.T) {
	c := testBootQuery(t)
	// Find an approved comment on post 1 if present.
	rows, err := wp.ListApprovedComments(c, 1)
	if err != nil || len(rows) == 0 {
		t.Skip("no approved comments on post 1")
	}
	cm := rows[0]
	state := &wp.CommentClassState{Depth: 1}
	c.Hooks.AddFilter("comment_class", func(value any, args ...any) any {
		list, _ := value.([]string)
		return append(list, "filtered-comment-class")
	}, 10, 5)
	classes := wp.GetCommentClass(c, cm.ID, 1, state)
	if !strings.Contains(strings.Join(classes, " "), "filtered-comment-class") {
		t.Fatalf("expected comment_class filter: %v", classes)
	}
	if !strings.Contains(strings.Join(classes, " "), "depth-1") {
		t.Fatalf("expected depth-1: %v", classes)
	}
}

func TestSanitizeHTMLClass(t *testing.T) {
	if got := wp.SanitizeHTMLClass("Foo Bar!", "x"); got != "FooBar" {
		t.Fatalf("SanitizeHTMLClass=%q", got)
	}
	if got := wp.SanitizeHTMLClass("!!!", "fallback_1"); got != "fallback_1" {
		t.Fatalf("fallback=%q", got)
	}
}
