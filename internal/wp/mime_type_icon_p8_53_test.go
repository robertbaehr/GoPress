package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.53: MimeTypeIcon for mime string and attachment ID.
func TestMimeTypeIconP853(t *testing.T) {
	c := mediaBoot(t)
	got := wp.MimeTypeIcon(c, "audio/mpeg", ".png")
	if !strings.Contains(got, "/wp-includes/images/media/audio.png") {
		t.Fatalf("audio=%q", got)
	}
	img := wp.MimeTypeIcon(c, "5", ".png")
	if !strings.Contains(img, "/images/media/") || !strings.HasSuffix(img, ".png") {
		t.Fatalf("attachment 5=%q", img)
	}
	def := wp.MimeTypeIcon(c, "application/octet-stream", "")
	if !strings.HasSuffix(def, "/default.png") {
		t.Fatalf("default=%q", def)
	}
	if wp.MimeTypeIcon(nil, "", ".png") != "" {
		t.Fatal("empty")
	}
}

func TestMimeTypeIconFilterP853(t *testing.T) {
	c := mediaBoot(t)
	c.Hooks.AddFilter("wp_mime_type_icon", func(value any, _ ...any) any {
		return "https://cdn.example/icon.svg"
	}, 10, 1)
	if got := wp.MimeTypeIcon(c, "video/mp4", ".svg"); got != "https://cdn.example/icon.svg" {
		t.Fatalf("filter=%q", got)
	}
}
