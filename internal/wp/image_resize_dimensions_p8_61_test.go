package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.61: ImageResizeDimensions scale-to-fit and center crop.
func TestImageResizeDimensionsP861(t *testing.T) {
	c := testBootQuery(t)
	r, ok := wp.ImageResizeDimensions(c, 800, 600, 400, 400, false)
	if !ok || r.DstW != 400 || r.DstH != 300 {
		t.Fatalf("scale=%+v ok=%v", r, ok)
	}
	r, ok = wp.ImageResizeDimensions(c, 800, 600, 400, 400, true)
	if !ok || r.DstW != 400 || r.DstH != 400 {
		t.Fatalf("crop=%+v ok=%v", r, ok)
	}
	if r.SrcW != 600 || r.SrcH != 600 {
		t.Fatalf("crop source box=%dx%d", r.SrcW, r.SrcH)
	}
	_, ok = wp.ImageResizeDimensions(c, 100, 100, 200, 200, false)
	if ok {
		t.Fatal("upscale should fail")
	}
}
