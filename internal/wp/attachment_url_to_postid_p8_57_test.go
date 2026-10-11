package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.57: AttachmentURLToPostID resolves fixture upload URL.
func TestAttachmentURLToPostIDP857(t *testing.T) {
	c := testBootQuery(t)
	url := wp.GetAttachmentURL(c, 5)
	if url == "" {
		t.Fatal("empty attachment URL")
	}
	got := wp.AttachmentURLToPostID(c, url)
	if got != 5 {
		t.Fatalf("id=%d want 5 url=%q", got, url)
	}
	if got := wp.AttachmentURLToPostID(c, "https://example.test/nope.jpg"); got != 0 {
		t.Fatalf("missing=%d", got)
	}
}

// P8.57: pre_attachment_url_to_postid short-circuit + attachment_url_to_postid filter.
func TestAttachmentURLToPostIDFiltersP857(t *testing.T) {
	c := testBootQuery(t)
	c.Hooks.AddFilter("pre_attachment_url_to_postid", func(value any, _ ...any) any {
		return int64(99)
	}, 10, 2)
	if got := wp.AttachmentURLToPostID(c, "https://ignored.test/x.jpg"); got != 99 {
		t.Fatalf("pre=%d", got)
	}

	c2 := testBootQuery(t)
	c2.Hooks.AddFilter("pre_attachment_url_to_postid", func(value any, _ ...any) any {
		return int64(0)
	}, 10, 2)
	if got := wp.AttachmentURLToPostID(c2, wp.GetAttachmentURL(c2, 5)); got != 0 {
		t.Fatalf("pre zero should short-circuit, got %d", got)
	}

	c3 := testBootQuery(t)
	c3.Hooks.AddFilter("attachment_url_to_postid", func(value any, _ ...any) any {
		return int64(42)
	}, 10, 2)
	if got := wp.AttachmentURLToPostID(c3, wp.GetAttachmentURL(c3, 5)); got != 42 {
		t.Fatalf("filter=%d", got)
	}
}
