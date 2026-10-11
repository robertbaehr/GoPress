package wp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// fallbackPDFPreviewSizes are the default fallback_intermediate_image_sizes (P8.34).
var fallbackPDFPreviewSizes = []string{"thumbnail", "medium", "large"}

// isPDFFile reports whether path looks like a PDF (extension or %PDF magic).
func isPDFFile(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if strings.EqualFold(filepath.Ext(path), ".pdf") {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var hdr [5]byte
	n, _ := f.Read(hdr[:])
	return n >= 4 && string(hdr[:4]) == "%PDF"
}

// renderPDFPreviewJPEG renders the first page of a PDF to a unique *-pdf.jpg beside it (P8.34).
// Uses Ghostscript when available (Imagick/GS path in WP). Returns preview path or "".
func renderPDFPreviewJPEG(pdfPath string) string {
	pdfPath = strings.TrimSpace(pdfPath)
	if pdfPath == "" {
		return ""
	}
	if _, err := exec.LookPath("gs"); err != nil {
		return ""
	}
	dir := filepath.Dir(pdfPath)
	ext := filepath.Ext(pdfPath)
	base := filepath.Base(pdfPath)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	previewName := UniqueFilename(dir, base+"-pdf.jpg")
	previewPath := filepath.Join(dir, previewName)

	// 128 DPI matches common WP Imagick PDF rasterization for letter-sized docs.
	cmd := exec.Command(
		"gs",
		"-q",
		"-dNOPAUSE",
		"-dBATCH",
		"-sDEVICE=jpeg",
		"-dFirstPage=1",
		"-dLastPage=1",
		"-r128",
		"-dJPEGQ=90",
		"-sOutputFile="+previewPath,
		pdfPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(previewPath)
		_ = out
		return ""
	}
	if st, err := os.Stat(previewPath); err != nil || st.Size() == 0 {
		_ = os.Remove(previewPath)
		return ""
	}
	return previewPath
}

// attachPDFPreviewSizes adds PDF preview full + intermediate sizes to metadata (P8.34).
// Mirrors wp_generate_attachment_metadata's application/pdf branch.
func attachPDFPreviewSizes(c *Context, meta map[string]any, pdfPath string) {
	if meta == nil || !isPDFFile(pdfPath) {
		return
	}
	preview := renderPDFPreviewJPEG(pdfPath)
	if preview == "" {
		return
	}
	info := GetImagesize(preview)
	if info == nil {
		return
	}
	fullBag := map[string]any{
		"file":      filepath.Base(preview),
		"width":     info.Width,
		"height":    info.Height,
		"mime-type": "image/jpeg",
	}
	if fs := Filesize(preview); fs > 0 {
		fullBag["filesize"] = fs
	}
	sizes := map[string]any{
		"full": fullBag,
	}

	fallback := append([]string(nil), fallbackPDFPreviewSizes...)
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("fallback_intermediate_image_sizes", fallback, meta)
		switch t := filtered.(type) {
		case []string:
			fallback = t
		case []any:
			fallback = fallback[:0]
			for _, item := range t {
				s := strings.TrimSpace(fmt.Sprint(item))
				if s != "" && s != "<nil>" {
					fallback = append(fallback, s)
				}
			}
		}
	}

	registered := GetRegisteredImageSubsizes(c)
	for _, name := range fallback {
		sz, ok := registered[name]
		if !ok {
			continue
		}
		// WP forces soft crops for PDF thumbnails.
		crop := sz.Crop
		if name == "thumbnail" {
			crop = false
		}
		made := MakeIntermediateSize(preview, sz.Width, sz.Height, crop)
		if made == nil {
			continue
		}
		sizes[name] = made
	}
	meta["sizes"] = sizes
}
