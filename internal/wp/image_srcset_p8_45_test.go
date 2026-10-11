package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.45: medium srcset includes medium + full; thumbnail crop has no srcset.
func TestGetAttachmentImageSrcsetMediumP845(t *testing.T) {
	c := mediaBoot(t)
	srcset := wp.GetAttachmentImageSrcset(c, 5, "medium", nil)
	if srcset == "" {
		t.Fatal("expected medium srcset")
	}
	if !strings.Contains(srcset, "sample-photo-a-300x225.jpg 300w") {
		t.Fatalf("missing medium candidate: %q", srcset)
	}
	if !strings.Contains(srcset, "sample-photo-a.jpg 640w") {
		t.Fatalf("missing full candidate: %q", srcset)
	}
	// Cropped thumbnail must not appear in medium srcset.
	if strings.Contains(srcset, "150x150") {
		t.Fatalf("thumbnail crop should be excluded: %q", srcset)
	}

	if got := wp.GetAttachmentImageSrcset(c, 5, "thumbnail", nil); got != "" {
		t.Fatalf("thumbnail-only ratio should yield no srcset, got %q", got)
	}
}

func TestGetAttachmentImageIncludesSrcsetP845(t *testing.T) {
	c := mediaBoot(t)
	html := wp.GetAttachmentImage(c, 5, "medium", false, nil)
	for _, want := range []string{
		`srcset=`,
		`300w`,
		`640w`,
		`sizes="auto, (max-width: 300px) 100vw, 300px"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
}

func TestCalculateImageSrcsetFilterP845(t *testing.T) {
	c := mediaBoot(t)
	c.Hooks.AddFilter("max_srcset_image_width", func(value any, _ ...any) any {
		return 400
	}, 10, 2)
	srcset := wp.GetAttachmentImageSrcset(c, 5, "medium", nil)
	// Full 640w exceeds max unless it is the src; medium is src so 640 excluded.
	if strings.Contains(srcset, "640w") {
		t.Fatalf("max width should exclude full: %q", srcset)
	}
	// With only one source left, srcset should be empty.
	if srcset != "" {
		t.Fatalf("single source should yield empty srcset: %q", srcset)
	}
}
