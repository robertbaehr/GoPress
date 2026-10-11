package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.49: GetAttachmentCaption reads post_excerpt for attachments.
func TestGetAttachmentCaptionP849(t *testing.T) {
	c := mediaBoot(t)
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET post_excerpt = %s WHERE ID = %d AND post_type = %s",
		"Fixture caption", int64(5), "attachment",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		q, args, _ := c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET post_excerpt = %s WHERE ID = %d",
			"", int64(5),
		)
		_, _ = c.DB.Exec(q, args...)
	})

	got := wp.GetAttachmentCaption(c, 5)
	if got != "Fixture caption" {
		t.Fatalf("caption=%q", got)
	}
	if got := wp.GetAttachmentCaption(c, 0); got != "" {
		t.Fatalf("zero id=%q", got)
	}
	if got := wp.GetAttachmentCaption(nil, 5); got != "" {
		t.Fatalf("nil ctx=%q", got)
	}
	// Non-attachment post ID should not return a caption.
	if got := wp.GetAttachmentCaption(c, 1); got != "" {
		t.Fatalf("non-attachment=%q", got)
	}
}

// P8.49: wp_get_attachment_caption filter can replace caption.
func TestGetAttachmentCaptionFilterP849(t *testing.T) {
	c := mediaBoot(t)
	c.Hooks.AddFilter("wp_get_attachment_caption", func(value any, _ ...any) any {
		return "Filtered caption"
	}, 10, 2)
	if got := wp.GetAttachmentCaption(c, 5); got != "Filtered caption" {
		t.Fatalf("filtered=%q", got)
	}
}
