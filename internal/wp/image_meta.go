// wp_read_image_metadata / attachment image_meta (P8.25 / P8.26).
package wp

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ImageMeta is the normalized EXIF/IPTC bag stored under attachment metadata["image_meta"].
type ImageMeta struct {
	Aperture         float64
	Credit           string
	Camera           string
	Caption          string
	CreatedTimestamp int64
	Copyright        string
	FocalLength      float64
	ISO              float64
	ShutterSpeed     float64
	Title            string
	Orientation      int
	Keywords         []string
	Alt              string
}

// emptyImageMetaMap returns the default wp_read_image_metadata field bag.
// keywords uses an empty PHP-serializable array (map[string]any{}).
func emptyImageMetaMap() map[string]any {
	return map[string]any{
		"aperture":          0.0,
		"credit":            "",
		"camera":            "",
		"caption":           "",
		"created_timestamp": int64(0),
		"copyright":         "",
		"focal_length":      0.0,
		"iso":               0.0,
		"shutter_speed":     0.0,
		"title":             "",
		"orientation":       0,
		"keywords":          map[string]any{},
		"alt":               "",
	}
}

// ReadImageMetadata is wp_read_image_metadata (P8.25 / P8.33).
// Returns nil,false when the file is missing or not an image.
// P8.33: JPEG EXIF APP1 is parsed for orientation/camera/exposure fields;
// remaining formats keep the empty bag. Filtered via wp_read_image_metadata.
func ReadImageMetadata(c *Context, file string) (map[string]any, bool) {
	file = strings.TrimSpace(file)
	if file == "" {
		return nil, false
	}
	info := GetImagesize(file)
	if info == nil {
		return nil, false
	}
	meta := emptyImageMetaMap()

	// P8.25 / P8.33: types filter (default JPEG + TIFF family).
	exifTypes := []any{ImageTypeJPEG, 7 /* TIFF_II */, 8 /* TIFF_MM */}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_read_image_metadata_types", exifTypes); v != nil {
			if arr, ok := v.([]any); ok {
				exifTypes = arr
			}
		}
	}
	if info.Type == ImageTypeJPEG && exifTypesAllow(exifTypes, ImageTypeJPEG) {
		applyJPEGExif(meta, file)
	}

	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_read_image_metadata", meta, file, info.Type)
		switch t := filtered.(type) {
		case map[string]any:
			meta = t
		case bool:
			if !t {
				return nil, false
			}
		case nil:
			return nil, false
		}
	}
	if meta == nil {
		return nil, false
	}
	// Ensure keywords is present (empty PHP array when unset).
	if _, ok := meta["keywords"]; !ok {
		meta["keywords"] = map[string]any{}
	}
	return meta, true
}

func exifTypesAllow(types []any, want int) bool {
	if len(types) == 0 {
		return true
	}
	for _, t := range types {
		switch v := t.(type) {
		case int:
			if v == want {
				return true
			}
		case int64:
			if int(v) == want {
				return true
			}
		case float64:
			if int(v) == want {
				return true
			}
		}
	}
	return false
}

// Filesize returns the file size in bytes without filters (wp_filesize core, P8.26).
// Returns 0 when the path is missing or unreadable.
func Filesize(file string) int64 {
	file = strings.TrimSpace(file)
	if file == "" {
		return 0
	}
	st, err := os.Stat(file)
	if err != nil || st.IsDir() {
		return 0
	}
	return st.Size()
}

// FilesizeFiltered mirrors wp_filesize() including pre_wp_filesize / wp_filesize
// filters (P2.117). Non-negative pre_wp_filesize short-circuits the os.Stat call.
// Negative / non-numeric pre values are ignored; the final filter result is max(0, n).
func FilesizeFiltered(c *Context, file string) int64 {
	file = strings.TrimSpace(file)
	if c != nil && c.Hooks != nil {
		pre := c.Hooks.ApplyFilters("pre_wp_filesize", nil, file)
		if n, ok := filesizeFilterInt(pre); ok && n >= 0 {
			return n
		}
	}
	size := Filesize(file)
	if c == nil || c.Hooks == nil {
		return size
	}
	filtered := c.Hooks.ApplyFilters("wp_filesize", size, file)
	if n, ok := filesizeFilterInt(filtered); ok {
		if n < 0 {
			return 0
		}
		return n
	}
	return 0
}

func filesizeFilterInt(v any) (int64, bool) {
	switch t := v.(type) {
	case nil:
		return 0, false
	case int:
		return int64(t), true
	case int32:
		return int64(t), true
	case int64:
		return t, true
	case uint:
		return int64(t), true
	case uint32:
		return int64(t), true
	case uint64:
		if t > uint64(^uint64(0)>>1) {
			return 0, true
		}
		return int64(t), true
	case float64:
		return int64(t), true
	case float32:
		return int64(t), true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

// GenerateAttachmentMetadata builds a minimal wp_generate_attachment_metadata bag (P8.25 / P8.26).
// For images: width/height/file/sizes{}/image_meta/filesize.
// Non-images: file + filesize when the path exists.
func GenerateAttachmentMetadata(c *Context, attachmentID int64, file string) map[string]any {
	return generateAttachmentMetadata(c, attachmentID, file, "create")
}

// UpdateAttachmentMetadata regenerates metadata with context "update" (P8.26).
func UpdateAttachmentMetadata(c *Context, attachmentID int64, file string) map[string]any {
	return generateAttachmentMetadata(c, attachmentID, file, "update")
}

func generateAttachmentMetadata(c *Context, attachmentID int64, file, context string) map[string]any {
	out := map[string]any{}
	file = strings.TrimSpace(file)
	if file == "" {
		return out
	}
	if context == "" {
		context = "create"
	}
	rel := file
	if c != nil {
		uploads := GetUploadDir(c)
		if uploads.Basedir != "" {
			if abs, err := filepath.Abs(file); err == nil {
				base := filepath.Clean(uploads.Basedir)
				if strings.HasPrefix(abs, base+string(filepath.Separator)) || abs == base {
					rel = filepath.ToSlash(strings.TrimPrefix(abs, base+string(filepath.Separator)))
				}
			}
		}
	}
	// Prefer basename path relative to uploads when already relative.
	if !filepath.IsAbs(rel) {
		rel = filepath.ToSlash(rel)
	} else {
		rel = filepath.ToSlash(filepath.Base(file))
	}

	info := GetImagesize(file)
	if info != nil {
		out["width"] = info.Width
		out["height"] = info.Height
		out["file"] = rel
		out["sizes"] = map[string]any{}
		if im, ok := ReadImageMetadata(c, file); ok {
			out["image_meta"] = im
		} else {
			out["image_meta"] = emptyImageMetaMap()
		}
		// P8.37: scale oversized originals to *-scaled.* (big_image_size_threshold).
		file = maybeScaleBigImage(c, attachmentID, file, out)
	} else {
		out["file"] = rel
		// P8.34: application/pdf → Ghostscript preview JPEG + fallback sizes.
		if isPDFFile(file) {
			attachPDFPreviewSizes(c, out, file)
		}
	}
	// P8.26 / P2.117: always capture filesize when the file exists (WP 6.0+).
	// Prefer scaled path filesize when maybeScaleBigImage already set it.
	// Uses FilesizeFiltered so pre_wp_filesize / wp_filesize hooks apply.
	if out["filesize"] == nil {
		if fs := FilesizeFiltered(c, file); fs > 0 {
			out["filesize"] = fs
		}
	}

	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_generate_attachment_metadata", out, attachmentID, context)
		if m, ok := filtered.(map[string]any); ok {
			out = m
		}
	}
	return out
}

// StoreAttachmentMetadata serializes and saves _wp_attachment_metadata (P8.25).
func StoreAttachmentMetadata(c *Context, attachmentID int64, meta map[string]any) error {
	if c == nil || c.Meta == nil || attachmentID <= 0 || meta == nil {
		return fmt.Errorf("wp: StoreAttachmentMetadata: invalid args")
	}
	serialized, err := SerializePHP(meta)
	if err != nil {
		return err
	}
	_, _, _, err = c.Meta.UpdateMetadata(MetaPost, attachmentID, "_wp_attachment_metadata", serialized, "")
	return err
}

// WPUpdateAttachmentMetadata is wp_update_attachment_metadata (P8.39).
// Applies the wp_update_attachment_metadata filter; empty/nil data deletes the
// meta key (PHP if ( $data )). Returns false when the attachment is missing
// or the write fails; true on success.
func WPUpdateAttachmentMetadata(c *Context, attachmentID int64, data map[string]any) bool {
	if c == nil || c.Meta == nil || attachmentID <= 0 {
		return false
	}
	post, err := GetPostByID(c, attachmentID)
	if err != nil || post == nil {
		return false
	}
	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_update_attachment_metadata", data, post.ID)
		switch t := filtered.(type) {
		case map[string]any:
			data = t
		case nil:
			data = nil
		default:
			// Non-map truthy values are unexpected; treat as no-op keep.
		}
	}
	if !attachmentMetadataTruthy(data) {
		ok, err := c.Meta.DeleteMetadata(MetaPost, post.ID, "_wp_attachment_metadata", "")
		return err == nil && ok
	}
	if err := StoreAttachmentMetadata(c, post.ID, data); err != nil {
		return false
	}
	return true
}

// attachmentMetadataTruthy mirrors PHP if ( $data ) for metadata maps (P8.39).
// nil or empty maps are falsy; non-empty maps are truthy.
func attachmentMetadataTruthy(data map[string]any) bool {
	return data != nil && len(data) > 0
}

// GenerateAndUpdateAttachmentMetadata generates metadata then persists via
// wp_update_attachment_metadata (P8.39).
func GenerateAndUpdateAttachmentMetadata(c *Context, attachmentID int64, file string) bool {
	meta := GenerateAttachmentMetadata(c, attachmentID, file)
	return WPUpdateAttachmentMetadata(c, attachmentID, meta)
}
