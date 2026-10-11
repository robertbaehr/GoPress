// Image editor subset (P8.30) — WP_Image_Editor-style resize/crop via stdlib.
package wp

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	_ "image/gif" // register GIF decoder
)

// ImageEditor is a minimal WP_Image_Editor stand-in loaded from a filesystem path.
type ImageEditor struct {
	path   string
	img    image.Image
	format string // "jpeg", "png", "gif"
}

// GetImageEditor loads an image for editing (wp_get_image_editor subset, P8.30).
func GetImageEditor(path string) (*ImageEditor, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("wp: GetImageEditor: empty path")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("wp: GetImageEditor: %w", err)
	}
	defer f.Close()
	img, format, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("wp: GetImageEditor: decode: %w", err)
	}
	switch format {
	case "jpeg", "png", "gif":
	default:
		return nil, fmt.Errorf("wp: GetImageEditor: unsupported format %q", format)
	}
	return &ImageEditor{path: path, img: img, format: format}, nil
}

// Size returns current width/height.
func (e *ImageEditor) Size() (w, h int) {
	if e == nil || e.img == nil {
		return 0, 0
	}
	b := e.img.Bounds()
	return b.Dx(), b.Dy()
}

// Crop extracts a source rectangle and optionally resamples to dstW×dstH
// (WP_Image_Editor::crop, P8.35). When srcAbs is true, srcW/srcH are treated as
// absolute bottom-right coordinates (Imagick/GD parity). Zero dst dimensions
// keep the cropped source size.
func (e *ImageEditor) Crop(srcX, srcY, srcW, srcH, dstW, dstH int, srcAbs bool) error {
	if e == nil || e.img == nil {
		return fmt.Errorf("wp: ImageEditor.Crop: nil editor")
	}
	if srcAbs {
		srcW -= srcX
		srcH -= srcY
	}
	if srcW <= 0 || srcH <= 0 {
		return fmt.Errorf("wp: ImageEditor.Crop: invalid source size")
	}
	imgW, imgH := e.Size()
	if srcX < 0 || srcY < 0 || srcX+srcW > imgW || srcY+srcH > imgH {
		return fmt.Errorf("wp: ImageEditor.Crop: source rect out of bounds")
	}
	if dstW <= 0 {
		dstW = srcW
	}
	if dstH <= 0 {
		dstH = srcH
	}
	cropped := cropImage(e.img, srcX, srcY, srcW, srcH)
	if dstW == srcW && dstH == srcH {
		e.img = cropped
		return nil
	}
	e.img = scaleImage(cropped, dstW, dstH)
	return nil
}

// Flip mirrors the image (WP_Image_Editor::flip, P8.38).
// horz flips along the horizontal axis (upside-down); vert along the vertical axis (left-right).
func (e *ImageEditor) Flip(horz, vert bool) error {
	if e == nil || e.img == nil {
		return fmt.Errorf("wp: ImageEditor.Flip: nil editor")
	}
	if !horz && !vert {
		return nil
	}
	e.img = flipImage(e.img, horz, vert)
	return nil
}

// Rotate rotates the image counter-clockwise by angle degrees (WP_Image_Editor::rotate, P8.36).
// Supports quarter-turns (0/90/180/270 after normalization); other angles error.
func (e *ImageEditor) Rotate(angle float64) error {
	if e == nil || e.img == nil {
		return fmt.Errorf("wp: ImageEditor.Rotate: nil editor")
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return fmt.Errorf("wp: ImageEditor.Rotate: invalid angle")
	}
	// Normalize to [0, 360).
	a := math.Mod(angle, 360)
	if a < 0 {
		a += 360
	}
	// Snap near-quarter turns (Imagick/GD media UI uses 90° steps).
	quarter := int(math.Round(a/90.0)) % 4
	if math.Abs(a-float64(quarter*90)) > 0.5 && !(quarter == 0 && math.Abs(a-360) <= 0.5) {
		return fmt.Errorf("wp: ImageEditor.Rotate: unsupported angle %v (quarter-turns only)", angle)
	}
	switch quarter {
	case 0:
		return nil
	case 1:
		e.img = rotateImage90CCW(e.img)
	case 2:
		e.img = rotateImage180(e.img)
	case 3:
		e.img = rotateImage270CCW(e.img)
	}
	return nil
}

// Resize scales (and optionally center-crops) to width×height (WP_Image_Editor::resize).
// Zero width or height means “preserve aspect for that axis”.
func (e *ImageEditor) Resize(width, height int, crop bool) error {
	if e == nil || e.img == nil {
		return fmt.Errorf("wp: ImageEditor.Resize: nil editor")
	}
	if width <= 0 && height <= 0 {
		return fmt.Errorf("wp: ImageEditor.Resize: empty dimensions")
	}
	srcW, srcH := e.Size()
	if srcW <= 0 || srcH <= 0 {
		return fmt.Errorf("wp: ImageEditor.Resize: invalid source")
	}
	dstW, dstH := constrainDimensions(srcW, srcH, width, height, crop)
	if dstW <= 0 || dstH <= 0 {
		return fmt.Errorf("wp: ImageEditor.Resize: computed empty size")
	}
	if !crop {
		e.img = scaleImage(e.img, dstW, dstH)
		return nil
	}
	// Crop: scale so the shorter side fills, then center-crop to target.
	scaleW := float64(dstW) / float64(srcW)
	scaleH := float64(dstH) / float64(srcH)
	scale := scaleW
	if scaleH > scaleW {
		scale = scaleH
	}
	scaledW := int(float64(srcW)*scale + 0.5)
	scaledH := int(float64(srcH)*scale + 0.5)
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}
	scaled := scaleImage(e.img, scaledW, scaledH)
	x0 := (scaledW - dstW) / 2
	y0 := (scaledH - dstH) / 2
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	e.img = cropImage(scaled, x0, y0, dstW, dstH)
	return nil
}

// Save writes the current image next to the source (or destPath when set).
// Returns path, width, height, mime-ish format (WP_Image_Editor::save subset).
func (e *ImageEditor) Save(destPath string) (string, int, int, error) {
	if e == nil || e.img == nil {
		return "", 0, 0, fmt.Errorf("wp: ImageEditor.Save: nil editor")
	}
	out := strings.TrimSpace(destPath)
	if out == "" {
		ext := filepath.Ext(e.path)
		base := strings.TrimSuffix(e.path, ext)
		w, h := e.Size()
		out = fmt.Sprintf("%s-%dx%d%s", base, w, h, ext)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", 0, 0, err
	}
	f, err := os.Create(out)
	if err != nil {
		return "", 0, 0, err
	}
	defer f.Close()
	switch e.format {
	case "png":
		err = png.Encode(f, e.img)
	case "gif":
		// Encode GIF as PNG-compatible path when gif encode is awkward; prefer PNG bytes
		// only if destination suggests png — otherwise re-encode as JPEG for simplicity.
		if strings.EqualFold(filepath.Ext(out), ".gif") {
			// stdlib gif.Encode needs paletted; fall back to PNG sibling.
			out = strings.TrimSuffix(out, filepath.Ext(out)) + ".png"
			_ = f.Close()
			f2, err2 := os.Create(out)
			if err2 != nil {
				return "", 0, 0, err2
			}
			defer f2.Close()
			err = png.Encode(f2, e.img)
			e.format = "png"
		} else {
			err = png.Encode(f, e.img)
			e.format = "png"
		}
	default:
		err = jpeg.Encode(f, e.img, &jpeg.Options{Quality: 82})
		e.format = "jpeg"
	}
	if err != nil {
		return "", 0, 0, err
	}
	w, h := e.Size()
	e.path = out
	return out, w, h, nil
}

// MakeIntermediateSize is image_make_intermediate_size without hooks (P8.30).
// Returns metadata bag {file,width,height,mime-type[,filesize]} or nil on failure.
func MakeIntermediateSize(file string, width, height int, crop bool) map[string]any {
	return MakeIntermediateSizeCtx(nil, file, width, height, crop)
}

// MakeIntermediateSizeCtx is image_make_intermediate_size (P8.30 / P8.40).
// Applies the image_make_intermediate_size filter to the saved path (basename
// of the filtered path becomes the `file` key), and records filesize when
// the saved file is readable.
func MakeIntermediateSizeCtx(c *Context, file string, width, height int, crop bool) map[string]any {
	if width <= 0 && height <= 0 {
		return nil
	}
	ed, err := GetImageEditor(file)
	if err != nil {
		return nil
	}
	if err := ed.Resize(width, height, crop); err != nil {
		return nil
	}
	path, w, h, err := ed.Save("")
	if err != nil || path == "" {
		return nil
	}
	filteredPath := path
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("image_make_intermediate_size", path); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				filteredPath = s
			}
		}
	}
	mime := "image/jpeg"
	switch ed.format {
	case "png":
		mime = "image/png"
	case "gif":
		mime = "image/gif"
	}
	meta := map[string]any{
		"file":      filepath.Base(filteredPath),
		"width":     w,
		"height":    h,
		"mime-type": mime,
	}
	// Prefer filesize of the real saved path (filter may only rename the basename).
	if st, err := os.Stat(path); err == nil {
		meta["filesize"] = int(st.Size())
	}
	return meta
}

// ResizeImage generates an intermediate file for size (P2.10 / P8.30).
func ResizeImage(srcPath string, size ImageSize) (string, error) {
	meta := MakeIntermediateSize(srcPath, size.Width, size.Height, size.Crop)
	if meta == nil {
		return "", fmt.Errorf("wp: ResizeImage: failed for %q", srcPath)
	}
	file, _ := meta["file"].(string)
	dir := filepath.Dir(srcPath)
	return filepath.Join(dir, file), nil
}

// ResizeImageCtx is ResizeImage with image_make_intermediate_size filter (P8.40).
func ResizeImageCtx(c *Context, srcPath string, size ImageSize) (string, error) {
	meta := MakeIntermediateSizeCtx(c, srcPath, size.Width, size.Height, size.Crop)
	if meta == nil {
		return "", fmt.Errorf("wp: ResizeImageCtx: failed for %q", srcPath)
	}
	file, _ := meta["file"].(string)
	dir := filepath.Dir(srcPath)
	return filepath.Join(dir, file), nil
}

func constrainDimensions(srcW, srcH, maxW, maxH int, crop bool) (int, int) {
	if crop && maxW > 0 && maxH > 0 {
		return maxW, maxH
	}
	if maxW <= 0 {
		maxW = srcW
	}
	if maxH <= 0 {
		maxH = srcH
	}
	// wp_constrain_dimensions subset: fit inside box preserving aspect.
	ratioW := float64(maxW) / float64(srcW)
	ratioH := float64(maxH) / float64(srcH)
	ratio := ratioW
	if ratioH < ratioW {
		ratio = ratioH
	}
	if ratio >= 1 {
		// Do not upscale.
		if maxW >= srcW && maxH >= srcH {
			return srcW, srcH
		}
	}
	dw := int(float64(srcW)*ratio + 0.5)
	dh := int(float64(srcH)*ratio + 0.5)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	return dw, dh
}

func scaleImage(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	srcW, srcH := sb.Dx(), sb.Dy()
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + y*srcH/h
		if sy >= sb.Max.Y {
			sy = sb.Max.Y - 1
		}
		for x := 0; x < w; x++ {
			sx := sb.Min.X + x*srcW/w
			if sx >= sb.Max.X {
				sx = sb.Max.X - 1
			}
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func cropImage(src image.Image, x, y, w, h int) image.Image {
	b := src.Bounds()
	r := image.Rect(b.Min.X+x, b.Min.Y+y, b.Min.X+x+w, b.Min.Y+y+h)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), src, r.Min, draw.Src)
	return dst
}

// flipImage mirrors along axes matching WP_Image_Editor_GD::flip (P8.38).
func flipImage(src image.Image, horz, vert bool) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := y
		if horz {
			sy = h - 1 - y
		}
		for x := 0; x < w; x++ {
			sx := x
			if vert {
				sx = w - 1 - x
			}
			dst.Set(x, y, src.At(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	return dst
}

// rotateImage90CCW maps (x,y) → (y, W-1-x); output size HxW.
func rotateImage90CCW(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(y, w-1-x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotateImage180(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(w-1-x, h-1-y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// rotateImage270CCW maps (x,y) → (H-1-y, x); output size HxW.
func rotateImage270CCW(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(h-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
