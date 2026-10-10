package wp_test

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestReadImageMetadataMissing(t *testing.T) {
	if _, ok := wp.ReadImageMetadata(nil, "/no/such/file.png"); ok {
		t.Fatal("missing file should be false")
	}
}

func TestReadImageMetadataDefaultsAndFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.png")
	writeTinyPNG(t, path)

	c := mediaBoot(t)
	meta, ok := wp.ReadImageMetadata(c, path)
	if !ok {
		t.Fatal("expected metadata")
	}
	if meta["aperture"] == nil || meta["keywords"] == nil {
		t.Fatalf("missing defaults: %#v", meta)
	}

	c.Hooks.AddFilter("wp_read_image_metadata", func(value any, args ...any) any {
		m, _ := value.(map[string]any)
		if m == nil {
			return value
		}
		m["camera"] = "GoPressCam"
		m["title"] = "Tiny"
		return m
	}, 10, 3)
	meta, ok = wp.ReadImageMetadata(c, path)
	if !ok || meta["camera"] != "GoPressCam" || meta["title"] != "Tiny" {
		t.Fatalf("filter not applied: %#v", meta)
	}
}

func TestGenerateAndStoreAttachmentMetadataOnUpload(t *testing.T) {
	c := mediaBoot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "upload-me.png")
	writeTinyPNG(t, path)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	up, err := wp.SaveUploadedMedia(c, "upload-me.png", f, 0)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	meta, ok := wp.GetAttachmentMetadata(c, up.ID)
	if !ok {
		t.Fatal("expected stored attachment metadata")
	}
	if meta["width"] == nil || meta["height"] == nil {
		t.Fatalf("missing dimensions: %#v", meta)
	}
	im, ok := meta["image_meta"].(map[string]any)
	if !ok {
		// may arrive as map[any]any from unserialize
		if _, ok2 := meta["image_meta"].(map[any]any); !ok2 {
			t.Fatalf("image_meta missing: %#v", meta)
		}
		return
	}
	if _, has := im["keywords"]; !has {
		t.Fatalf("image_meta.keywords missing: %#v", im)
	}
}

func writeTinyPNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
