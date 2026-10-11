// Big image scaling (P8.37) — big_image_size_threshold / -scaled full size.
package wp

import (
	"path/filepath"
	"strings"
)

// BigImageSizeThreshold returns the max edge length before scaling (default 2560).
// Returning 0 disables scaling. Filter: big_image_size_threshold (P8.37).
func BigImageSizeThreshold(c *Context, width, height int, file string, attachmentID int64) int {
	threshold := 2560
	if c == nil || c.Hooks == nil {
		return threshold
	}
	filtered := c.Hooks.ApplyFilters(
		"big_image_size_threshold",
		threshold,
		[]int{width, height},
		file,
		attachmentID,
	)
	switch t := filtered.(type) {
	case bool:
		if !t {
			return 0
		}
		return threshold
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case nil:
		return 0
	default:
		return threshold
	}
}

// maybeScaleBigImage scales oversized originals to *-scaled.* and records original_image (P8.37).
// Updates out width/height/file/filesize and _wp_attached_file when attachmentID > 0.
func maybeScaleBigImage(c *Context, attachmentID int64, file string, out map[string]any) string {
	if out == nil {
		return file
	}
	w, _ := out["width"].(int)
	h, _ := out["height"].(int)
	if w <= 0 {
		switch t := out["width"].(type) {
		case int64:
			w = int(t)
		case float64:
			w = int(t)
		}
	}
	if h <= 0 {
		switch t := out["height"].(type) {
		case int64:
			h = int(t)
		case float64:
			h = int(t)
		}
	}
	if w <= 0 || h <= 0 {
		return file
	}
	threshold := BigImageSizeThreshold(c, w, h, file, attachmentID)
	if threshold <= 0 || (w <= threshold && h <= threshold) {
		return file
	}
	ed, err := GetImageEditor(file)
	if err != nil {
		return file
	}
	if err := ed.Resize(threshold, threshold, false); err != nil {
		return file
	}
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	dest := base + "-scaled" + ext
	saved, nw, nh, err := ed.Save(dest)
	if err != nil || saved == "" {
		return file
	}
	out["width"] = nw
	out["height"] = nh
	out["original_image"] = filepath.Base(file)
	rel := filepath.ToSlash(filepath.Base(saved))
	if c != nil {
		uploads := GetUploadDir(c)
		if uploads.Basedir != "" {
			if abs, err := filepath.Abs(saved); err == nil {
				baseDir := filepath.Clean(uploads.Basedir)
				if strings.HasPrefix(abs, baseDir+string(filepath.Separator)) {
					rel = filepath.ToSlash(strings.TrimPrefix(abs, baseDir+string(filepath.Separator)))
				}
			}
		}
	}
	out["file"] = rel
	if fs := Filesize(saved); fs > 0 {
		out["filesize"] = fs
	}
	if c != nil && c.Meta != nil && attachmentID > 0 {
		_, _, _, _ = c.Meta.UpdateMetadata(MetaPost, attachmentID, "_wp_attached_file", rel, "")
	}
	return saved
}
