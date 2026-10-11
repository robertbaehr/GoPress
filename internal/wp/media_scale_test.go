package wp_test

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func writeLargePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestBigImageSizeThresholdDefaultAndFilter(t *testing.T) {
	c := mediaBoot(t)
	if got := wp.BigImageSizeThreshold(c, 3000, 2000, "x.png", 1); got != 2560 {
		t.Fatalf("default=%d", got)
	}
	c.Hooks.AddFilter("big_image_size_threshold", func(value any, args ...any) any {
		return false
	}, 10, 4)
	if got := wp.BigImageSizeThreshold(c, 3000, 2000, "x.png", 1); got != 0 {
		t.Fatalf("disabled=%d", got)
	}
}

func TestGenerateAttachmentMetadataScalesBigImage(t *testing.T) {
	c := mediaBoot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.png")
	// 2800x100 exceeds default 2560 threshold on width.
	writeLargePNG(t, path, 2800, 100)

	meta := wp.GenerateAttachmentMetadata(c, 0, path)
	orig, _ := meta["original_image"].(string)
	if orig != "huge.png" {
		t.Fatalf("original_image=%v meta=%#v", meta["original_image"], meta)
	}
	w, _ := meta["width"].(int)
	if w > 2560 {
		t.Fatalf("scaled width=%d want ≤2560", w)
	}
	file, _ := meta["file"].(string)
	if !filepath.IsAbs(file) && file != "huge-scaled.png" && filepath.Base(file) != "huge-scaled.png" {
		t.Fatalf("file=%q", file)
	}
	scaledPath := filepath.Join(dir, "huge-scaled.png")
	if _, err := os.Stat(scaledPath); err != nil {
		t.Fatalf("scaled file missing: %v", err)
	}
}
