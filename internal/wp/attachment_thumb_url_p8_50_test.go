package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.50: GetAttachmentThumbURL returns thumbnail size URL for attachments.
func TestGetAttachmentThumbURLP850(t *testing.T) {
	c := mediaBoot(t)
	got := wp.GetAttachmentThumbURL(c, 5)
	if got == "" {
		t.Fatal("expected thumb url")
	}
	if !strings.Contains(got, "http") {
		t.Fatalf("thumb=%q", got)
	}
	// Should match GetAttachmentImageURL thumbnail.
	want := wp.GetAttachmentImageURL(c, 5, "thumbnail", false)
	if got != want {
		t.Fatalf("thumb=%q imageURL=%q", got, want)
	}
	if got := wp.GetAttachmentThumbURL(c, 0); got != "" {
		t.Fatalf("zero id=%q", got)
	}
	if got := wp.GetAttachmentThumbURL(nil, 5); got != "" {
		t.Fatalf("nil ctx=%q", got)
	}
	if got := wp.GetAttachmentThumbURL(c, 1); got != "" {
		t.Fatalf("non-attachment=%q", got)
	}
}

// P8.50: wp_get_attachment_thumb_url filter can replace URL.
func TestGetAttachmentThumbURLFilterP850(t *testing.T) {
	c := mediaBoot(t)
	c.Hooks.AddFilter("wp_get_attachment_thumb_url", func(value any, _ ...any) any {
		return "https://cdn.example/thumb.jpg"
	}, 10, 2)
	if got := wp.GetAttachmentThumbURL(c, 5); got != "https://cdn.example/thumb.jpg" {
		t.Fatalf("filtered=%q", got)
	}
}
