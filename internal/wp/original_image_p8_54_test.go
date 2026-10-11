package wp_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.54: without original_image meta, path/URL match attached file / attachment URL.
func TestGetOriginalImageDefaultsP854(t *testing.T) {
	c := testBootQuery(t)
	path := wp.GetOriginalImagePath(c, 5)
	if path == "" || !strings.Contains(path, "sample-photo-a.jpg") {
		t.Fatalf("path=%q", path)
	}
	url := wp.GetOriginalImageURL(c, 5)
	if url == "" || !strings.Contains(url, "sample-photo-a.jpg") {
		t.Fatalf("url=%q", url)
	}
	if wp.GetOriginalImagePath(c, 10) != "" { // PDF
		t.Fatal("non-image should be empty")
	}
	if wp.GetOriginalImageURL(nil, 5) != "" {
		t.Fatal("nil context")
	}
}

// P8.54: original_image meta joins dirname of attached file/URL.
func TestGetOriginalImageFromMetaP854(t *testing.T) {
	c := testBootQuery(t)
	meta, ok := wp.GetAttachmentMetadata(c, 5)
	if !ok {
		meta = map[string]any{}
	}
	prev := meta["original_image"]
	meta["original_image"] = "sample-photo-a-original.jpg"
	if _, _, _, err := c.Meta.UpdateMetadata(wp.MetaPost, 5, "_wp_attachment_metadata", wp.MaybeSerialize(meta), ""); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if prev == nil {
			delete(meta, "original_image")
		} else {
			meta["original_image"] = prev
		}
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaPost, 5, "_wp_attachment_metadata", wp.MaybeSerialize(meta), "")
	})

	path := wp.GetOriginalImagePath(c, 5)
	if filepath.Base(path) != "sample-photo-a-original.jpg" {
		t.Fatalf("path=%q", path)
	}
	url := wp.GetOriginalImageURL(c, 5)
	if !strings.HasSuffix(url, "/sample-photo-a-original.jpg") {
		t.Fatalf("url=%q", url)
	}

	c.Hooks.AddFilter("wp_get_original_image_url", func(value any, _ ...any) any {
		return "https://cdn.example/orig.jpg"
	}, 10, 2)
	if got := wp.GetOriginalImageURL(c, 5); got != "https://cdn.example/orig.jpg" {
		t.Fatalf("filtered=%q", got)
	}
}
