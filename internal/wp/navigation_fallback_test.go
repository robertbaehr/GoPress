package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetOrCreateNavigationFallback(t *testing.T) {
	c := bootWP(t)
	post, err := wp.GetOrCreateNavigationFallback(c)
	if err != nil {
		t.Fatalf("fallback: %v", err)
	}
	if post == nil || post.ID <= 0 {
		t.Fatalf("expected navigation post, got %#v", post)
	}
	again, err := wp.GetOrCreateNavigationFallback(c)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if again == nil || again.ID != post.ID {
		t.Fatalf("expected same id %d, got %#v", post.ID, again)
	}
}
