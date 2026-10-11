package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.48: GetAttachmentImageSrcAny accepts [w,h] and picks a fitting size.
func TestGetAttachmentImageSrcArraySizeP848(t *testing.T) {
	c := mediaBoot(t)
	src, ok := wp.GetAttachmentImageSrcAny(c, 5, []int{150, 150}, false)
	if !ok {
		t.Fatal("array size")
	}
	if src.URL == "" || src.Width <= 0 || src.Height <= 0 {
		t.Fatalf("src=%+v", src)
	}
	// Fixture thumbnail is 150x150-ish; array request should not exceed.
	if src.Width > 150 || src.Height > 150 {
		t.Fatalf("expected constrained size, got %+v", src)
	}
}

// P8.48: string size path unchanged via GetAttachmentImageSrc.
func TestGetAttachmentImageSrcStringP848(t *testing.T) {
	c := mediaBoot(t)
	src, ok := wp.GetAttachmentImageSrc(c, 5, "medium", false)
	if !ok || src.URL == "" {
		t.Fatalf("medium=%+v ok=%v", src, ok)
	}
}

// P8.48: wp_get_attachment_image_src filter returning false clears result.
func TestGetAttachmentImageSrcFilterFalseP848(t *testing.T) {
	c := mediaBoot(t)
	c.Hooks.AddFilter("wp_get_attachment_image_src", func(value any, _ ...any) any {
		return false
	}, 10, 4)
	_, ok := wp.GetAttachmentImageSrc(c, 5, "thumbnail", false)
	if ok {
		t.Fatal("filter false should yield ok=false")
	}
}

// P8.48: filter can replace src array.
func TestGetAttachmentImageSrcFilterReplaceP848(t *testing.T) {
	c := mediaBoot(t)
	c.Hooks.AddFilter("wp_get_attachment_image_src", func(value any, _ ...any) any {
		return []any{"https://example.com/replaced.jpg", 320, 200, true}
	}, 10, 4)
	src, ok := wp.GetAttachmentImageSrc(c, 5, "thumbnail", false)
	if !ok {
		t.Fatal("expected ok")
	}
	if src.URL != "https://example.com/replaced.jpg" || src.Width != 320 || src.Height != 200 || !src.Resized {
		t.Fatalf("src=%+v", src)
	}
}
