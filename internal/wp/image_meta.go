// wp_read_image_metadata / attachment image_meta (P8.25 / P8.26).
package wp

import (
	"fmt"
	"os"
	"path/filepath"
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

// ReadImageMetadata is wp_read_image_metadata (P8.25).
// Returns nil,false when the file is missing or not an image.
// Full EXIF/IPTC parsing is stubbed; the default empty bag is returned and
// filtered via wp_read_image_metadata so callers can inject fields.
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

	// P8.25: optional type filter (wp_read_image_metadata_types) — accepted, unused without EXIF.
	if c != nil && c.Hooks != nil {
		_ = c.Hooks.ApplyFilters("wp_read_image_metadata_types", []any{
			ImageTypeJPEG,
		})
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

// Filesize returns the file size in bytes (wp_filesize, P8.26).
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
	} else {
		out["file"] = rel
	}
	// P8.26: always capture filesize when the file exists (WP 6.0+).
	if fs := Filesize(file); fs > 0 {
		out["filesize"] = fs
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
