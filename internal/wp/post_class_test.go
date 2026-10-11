package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetPostClassBasics(t *testing.T) {
	c := testBootQuery(t)
	classes := wp.GetPostClass(c, 1)
	joined := strings.Join(classes, " ")
	for _, want := range []string{"post-1", "post", "type-post", "status-publish", "format-standard", "hentry"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
}

func TestGetPostClassPasswordAndExtraAndFilter(t *testing.T) {
	c := testBootQuery(t)
	// Fixture post 19 has post_password=secret.
	classes := wp.GetPostClass(c, 19, "custom-extra another")
	joined := strings.Join(classes, " ")
	if !strings.Contains(joined, "post-password-required") {
		t.Fatalf("expected password class: %q", joined)
	}
	if !strings.Contains(joined, "custom-extra") || !strings.Contains(joined, "another") {
		t.Fatalf("expected css_class extras: %q", joined)
	}
	// Extras should appear before post-N.
	if i := strings.Index(joined, "custom-extra"); i < 0 || i > strings.Index(joined, "post-19") {
		t.Fatalf("extras should precede post-19: %q", joined)
	}

	c.Hooks.AddFilter("post_class", func(value any, args ...any) any {
		list, _ := value.([]string)
		return append(list, "filtered-post-class")
	}, 10, 3)
	filtered := wp.GetPostClass(c, 1)
	if !strings.Contains(strings.Join(filtered, " "), "filtered-post-class") {
		t.Fatalf("expected post_class filter: %v", filtered)
	}
}

func TestGetPostClassHasThumbnail(t *testing.T) {
	c := testBootQuery(t)
	// Attach a thumbnail meta on post 13 for this test, then clear.
	_, _, _, err := c.Meta.UpdateMetadata(wp.MetaPost, 13, "_thumbnail_id", "1", "")
	if err != nil {
		t.Fatalf("set thumbnail: %v", err)
	}
	t.Cleanup(func() {
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, 13, "_thumbnail_id", "")
	})
	classes := wp.GetPostClass(c, 13)
	if !strings.Contains(strings.Join(classes, " "), "has-post-thumbnail") {
		t.Fatalf("expected has-post-thumbnail: %v", classes)
	}
}
