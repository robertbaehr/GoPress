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
	if meta["filesize"] == nil {
		t.Fatalf("missing filesize: %#v", meta)
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

func TestGenerateAttachmentMetadataFilesizeAndUpdateContext(t *testing.T) {
	c := mediaBoot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.png")
	writeTinyPNG(t, path)

	var sawContext string
	c.Hooks.AddFilter("wp_generate_attachment_metadata", func(value any, args ...any) any {
		if len(args) >= 2 {
			if s, ok := args[1].(string); ok {
				sawContext = s
			}
		}
		return value
	}, 10, 3)

	meta := wp.GenerateAttachmentMetadata(c, 99, path)
	if meta["filesize"] == nil {
		t.Fatalf("expected filesize: %#v", meta)
	}
	fs, _ := meta["filesize"].(int64)
	if fs <= 0 {
		// may be int from JSON-ish paths
		switch v := meta["filesize"].(type) {
		case int:
			fs = int64(v)
		case float64:
			fs = int64(v)
		}
	}
	if fs <= 0 {
		t.Fatalf("filesize=%v", meta["filesize"])
	}
	if want := wp.Filesize(path); want != fs {
		t.Fatalf("filesize=%d want %d", fs, want)
	}
	if sawContext != "create" {
		t.Fatalf("create context=%q", sawContext)
	}

	_ = wp.UpdateAttachmentMetadata(c, 99, path)
	if sawContext != "update" {
		t.Fatalf("update context=%q", sawContext)
	}

	// Non-image: filesize + file only.
	txt := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(txt, []byte("hello gopress"), 0o644); err != nil {
		t.Fatal(err)
	}
	nonImg := wp.GenerateAttachmentMetadata(c, 1, txt)
	if nonImg["file"] == nil || nonImg["filesize"] == nil {
		t.Fatalf("non-image meta: %#v", nonImg)
	}
	if nonImg["width"] != nil {
		t.Fatalf("non-image should not have width: %#v", nonImg)
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

// P2.117: pre_wp_filesize short-circuit + wp_filesize filter.
func TestFilesizeFilteredHooks(t *testing.T) {
	c := mediaBoot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sized.bin")
	if err := os.WriteFile(path, []byte("abcdefghij"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := wp.Filesize(path); got != 10 {
		t.Fatalf("Filesize=%d want 10", got)
	}
	if got := wp.FilesizeFiltered(c, path); got != 10 {
		t.Fatalf("unfiltered FilesizeFiltered=%d want 10", got)
	}

	c.Hooks.AddFilter("pre_wp_filesize", func(value any, args ...any) any {
		return int64(42)
	}, 10, 2)
	if got := wp.FilesizeFiltered(c, path); got != 42 {
		t.Fatalf("pre_wp_filesize short-circuit=%d want 42", got)
	}
	if got := wp.FilesizeFiltered(c, "/no/such/file"); got != 42 {
		t.Fatalf("pre bypasses missing file: %d", got)
	}

	// Negative pre is ignored (WP 7.1+).
	c2 := mediaBoot(t)
	c2.Hooks.AddFilter("pre_wp_filesize", func(value any, args ...any) any {
		return int64(-1)
	}, 10, 2)
	c2.Hooks.AddFilter("wp_filesize", func(value any, args ...any) any {
		if n, ok := value.(int64); ok {
			return n + 5
		}
		return value
	}, 10, 2)
	if got := wp.FilesizeFiltered(c2, path); got != 15 {
		t.Fatalf("wp_filesize filter=%d want 15", got)
	}
}
