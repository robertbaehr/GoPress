package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.46: ImageSizeFromMeta + CalculateImageSizesFromName + sizes filter + auto prefix.
func TestImageSizeFromMetaP846(t *testing.T) {
	c := mediaBoot(t)
	meta, ok := wp.GetAttachmentMetadata(c, 5)
	if !ok {
		t.Fatal("expected attachment 5 metadata")
	}
	w, h, ok := wp.ImageSizeFromMeta("medium", meta)
	if !ok || w != 300 || h != 225 {
		t.Fatalf("medium: w=%d h=%d ok=%v", w, h, ok)
	}
	w, h, ok = wp.ImageSizeFromMeta("full", meta)
	if !ok || w != 640 {
		t.Fatalf("full: w=%d h=%d ok=%v", w, h, ok)
	}
	if _, _, ok = wp.ImageSizeFromMeta("missing-size", meta); ok {
		t.Fatal("missing size should fail")
	}
}

func TestCalculateImageSizesFromNameP846(t *testing.T) {
	c := mediaBoot(t)
	meta, _ := wp.GetAttachmentMetadata(c, 5)
	got := wp.CalculateImageSizesFromName(c, "medium", "http://example/x.jpg", meta, 5)
	want := "(max-width: 300px) 100vw, 300px"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if wp.CalculateImageSizesFromName(c, "nope", "", meta, 5) != "" {
		t.Fatal("unknown size")
	}
}

func TestGetAttachmentImageSizesFilterP846(t *testing.T) {
	c := mediaBoot(t)
	c.Hooks.AddFilter("wp_get_attachment_image_sizes", func(value any, args ...any) any {
		return "99vw"
	}, 10, 5)
	got := wp.GetAttachmentImageSizes(c, 5, "medium", nil)
	if got != "99vw" {
		t.Fatalf("filter: got %q", got)
	}
}

func TestCalculateImageSizesFromNameFilterStringP846(t *testing.T) {
	c := mediaBoot(t)
	meta, _ := wp.GetAttachmentMetadata(c, 5)
	var seenSize any
	c.Hooks.AddFilter("wp_calculate_image_sizes", func(value any, args ...any) any {
		if len(args) > 0 {
			seenSize = args[0]
		}
		return value
	}, 10, 5)
	_ = wp.CalculateImageSizesFromName(c, "thumbnail", "", meta, 5)
	if s, ok := seenSize.(string); !ok || s != "thumbnail" {
		t.Fatalf("filter size arg want string thumbnail, got %#v", seenSize)
	}
}

func TestPrependAutoSizesP846(t *testing.T) {
	c := mediaBoot(t)
	html := wp.GetAttachmentImage(c, 5, "medium", false, map[string]string{
		"sizes":   "(max-width: 300px) 100vw, 300px",
		"loading": "lazy",
	})
	if !strings.Contains(html, `sizes="auto, (max-width: 300px) 100vw, 300px"`) {
		t.Fatalf("auto prefix: %q", html)
	}
	// Already has auto — no double prefix.
	html = wp.GetAttachmentImage(c, 5, "medium", false, map[string]string{
		"sizes":   "auto, 50vw",
		"loading": "lazy",
		"srcset":  "x 1w, y 2w", // skip srcset generation path
	})
	if strings.Contains(html, "auto, auto") {
		t.Fatalf("double auto: %q", html)
	}
	// Non-lazy — no auto prefix.
	html = wp.GetAttachmentImage(c, 5, "medium", false, map[string]string{
		"sizes":   "(max-width: 300px) 100vw, 300px",
		"loading": "eager",
	})
	if strings.Contains(html, `sizes="auto,`) {
		t.Fatalf("eager should not get auto: %q", html)
	}
}
