package wp_test

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestMakeIntermediateSizeFilterAndFilesize(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wide.png")
	writePNGForIntermediate(t, src, 400, 200)

	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("image_make_intermediate_size", func(value any, _ ...any) any {
		path, _ := value.(string)
		return filepath.Join(filepath.Dir(path), "custom-intermediate.png")
	}, 10, 1)

	meta := wp.MakeIntermediateSizeCtx(c, src, 100, 0, false)
	if meta == nil {
		t.Fatal("nil meta")
	}
	file, _ := meta["file"].(string)
	if file != "custom-intermediate.png" {
		t.Fatalf("filtered file=%q", file)
	}
	if meta["width"] != 100 || meta["height"] != 50 {
		t.Fatalf("dims=%v x %v", meta["width"], meta["height"])
	}
	fs, ok := meta["filesize"].(int)
	if !ok || fs <= 0 {
		t.Fatalf("filesize=%v", meta["filesize"])
	}
	// Real file still written under dimension name; filter only renames metadata basename.
	if _, err := os.Stat(filepath.Join(dir, "wide-100x50.png")); err != nil {
		// Save naming may vary — ensure some png sibling exists.
		entries, _ := os.ReadDir(dir)
		found := false
		for _, e := range entries {
			if e.Name() != "wide.png" && strings.HasSuffix(e.Name(), ".png") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected resized file in %s: %v", dir, entries)
		}
	}
}

func TestMakeIntermediateSizeZeroDimsNil(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "x.png")
	writePNGForIntermediate(t, src, 10, 10)
	if wp.MakeIntermediateSizeCtx(nil, src, 0, 0, false) != nil {
		t.Fatal("expected nil for zero dims")
	}
}

func TestResizeImageCtxUsesFilter(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	writePNGForIntermediate(t, src, 200, 200)
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("image_make_intermediate_size", func(value any, _ ...any) any {
		return filepath.Join(dir, "thumb-alias.png")
	}, 10, 1)
	path, err := wp.ResizeImageCtx(c, src, wp.ImageSize{Name: "thumb", Width: 50, Height: 50, Crop: true})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "thumb-alias.png" {
		t.Fatalf("path=%q", path)
	}
}

func writePNGForIntermediate(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 40, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
