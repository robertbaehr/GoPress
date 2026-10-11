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

func TestGetImageEditorResizeCropSave(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "sample.png")
	writeTestPNG(t, src, 200, 100)

	ed, err := wp.GetImageEditor(src)
	if err != nil {
		t.Fatalf("GetImageEditor: %v", err)
	}
	w, h := ed.Size()
	if w != 200 || h != 100 {
		t.Fatalf("Size=%dx%d", w, h)
	}
	if err := ed.Resize(50, 50, true); err != nil {
		t.Fatalf("Resize crop: %v", err)
	}
	w, h = ed.Size()
	if w != 50 || h != 50 {
		t.Fatalf("after crop Size=%dx%d", w, h)
	}
	out, ow, oh, err := ed.Save("")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if ow != 50 || oh != 50 {
		t.Fatalf("saved dims %dx%d", ow, oh)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("missing output %q: %v", out, err)
	}
	if !strings.Contains(filepath.Base(out), "50x50") {
		t.Fatalf("expected sized filename, got %q", out)
	}
}

func TestImageEditorCrop(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "crop-src.png")
	writeTestPNG(t, src, 100, 80)

	ed, err := wp.GetImageEditor(src)
	if err != nil {
		t.Fatalf("GetImageEditor: %v", err)
	}
	if err := ed.Crop(10, 10, 40, 30, 0, 0, false); err != nil {
		t.Fatalf("Crop: %v", err)
	}
	w, h := ed.Size()
	if w != 40 || h != 30 {
		t.Fatalf("after Crop Size=%dx%d", w, h)
	}

	ed2, err := wp.GetImageEditor(src)
	if err != nil {
		t.Fatal(err)
	}
	// src_abs: srcW/srcH are bottom-right coords → width=50-10=40, height=50-10=40.
	if err := ed2.Crop(10, 10, 50, 50, 20, 20, true); err != nil {
		t.Fatalf("Crop abs: %v", err)
	}
	w, h = ed2.Size()
	if w != 20 || h != 20 {
		t.Fatalf("after abs Crop+scale Size=%dx%d", w, h)
	}

	ed3, err := wp.GetImageEditor(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := ed3.Crop(0, 0, 0, 10, 0, 0, false); err == nil {
		t.Fatal("expected error for empty crop width")
	}
}

func TestImageEditorFlip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "flip-src.png")
	writeTestPNG(t, src, 30, 20)

	ed, err := wp.GetImageEditor(src)
	if err != nil {
		t.Fatalf("GetImageEditor: %v", err)
	}
	if err := ed.Flip(false, false); err != nil {
		t.Fatalf("Flip none: %v", err)
	}
	w, h := ed.Size()
	if w != 30 || h != 20 {
		t.Fatalf("Size=%dx%d", w, h)
	}
	if err := ed.Flip(true, false); err != nil {
		t.Fatalf("Flip horz: %v", err)
	}
	w, h = ed.Size()
	if w != 30 || h != 20 {
		t.Fatalf("after horz Size=%dx%d", w, h)
	}
	if err := ed.Flip(false, true); err != nil {
		t.Fatalf("Flip vert: %v", err)
	}
	if err := ed.Flip(true, true); err != nil {
		t.Fatalf("Flip both: %v", err)
	}
	w, h = ed.Size()
	if w != 30 || h != 20 {
		t.Fatalf("after both Size=%dx%d", w, h)
	}
	dest := filepath.Join(dir, "flip-out.png")
	if _, _, _, err := ed.Save(dest); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func TestImageEditorRotate(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "rot-src.png")
	writeTestPNG(t, src, 40, 20)

	ed, err := wp.GetImageEditor(src)
	if err != nil {
		t.Fatalf("GetImageEditor: %v", err)
	}
	if err := ed.Rotate(90); err != nil {
		t.Fatalf("Rotate 90: %v", err)
	}
	w, h := ed.Size()
	if w != 20 || h != 40 {
		t.Fatalf("after 90° Size=%dx%d want 20x40", w, h)
	}
	if err := ed.Rotate(90); err != nil {
		t.Fatalf("Rotate 90 again: %v", err)
	}
	w, h = ed.Size()
	if w != 40 || h != 20 {
		t.Fatalf("after 180° Size=%dx%d want 40x20", w, h)
	}
	if err := ed.Rotate(180); err != nil {
		t.Fatalf("Rotate 180: %v", err)
	}
	w, h = ed.Size()
	if w != 40 || h != 20 {
		t.Fatalf("after +180 Size=%dx%d", w, h)
	}
	if err := ed.Rotate(0); err != nil {
		t.Fatalf("Rotate 0: %v", err)
	}
	if err := ed.Rotate(45); err == nil {
		t.Fatal("expected error for non-quarter angle")
	}
}

func TestMakeIntermediateSizeAndResizeImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wide.png")
	writeTestPNG(t, src, 400, 200)

	meta := wp.MakeIntermediateSize(src, 100, 0, false)
	if meta == nil {
		t.Fatal("MakeIntermediateSize returned nil")
	}
	if meta["width"] != 100 {
		t.Fatalf("width=%v", meta["width"])
	}
	// 400x200 → width 100 keeps aspect → height 50.
	if meta["height"] != 50 {
		t.Fatalf("height=%v want 50", meta["height"])
	}
	file, _ := meta["file"].(string)
	if file == "" || !strings.Contains(file, "100x50") {
		t.Fatalf("file=%q", file)
	}

	path, err := wp.ResizeImage(src, wp.ImageSize{Name: "thumb", Width: 80, Height: 80, Crop: true})
	if err != nil {
		t.Fatalf("ResizeImage: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("resized missing: %v", err)
	}
}

func writeTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 80, A: 255})
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
