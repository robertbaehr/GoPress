package wp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadImageToEditPath is _load_image_to_edit_path (P8.28).
// Returns the filesystem path (or filtered URL) for editing an attachment image.
// size defaults to "full"; intermediate sizes resolve via attachment metadata.
func LoadImageToEditPath(c *Context, attachmentID int64, size string) string {
	if attachmentID <= 0 {
		return ""
	}
	size = strings.TrimSpace(size)
	if size == "" {
		size = "full"
	}
	filepathStr := GetAttachedFile(c, attachmentID)
	if filepathStr != "" {
		if _, err := os.Stat(filepathStr); err == nil {
			if size != "full" {
				if data := ImageGetIntermediateSize(c, attachmentID, size); data != nil {
					file, _ := data["file"].(string)
					file = strings.TrimSpace(file)
					if file != "" {
						filepathStr = filepath.Join(filepath.Dir(filepathStr), file)
						if c != nil && c.Hooks != nil {
							if v := c.Hooks.ApplyFilters("load_image_to_edit_filesystempath", filepathStr, attachmentID, size); v != nil {
								if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
									filepathStr = s
								}
							}
						}
					}
				}
			}
		} else {
			filepathStr = ""
		}
	}
	if filepathStr == "" {
		// Remote/URL fallback when local file is missing (allow_url_fopen path).
		url := GetAttachmentURL(c, attachmentID)
		if c != nil && c.Hooks != nil {
			if v := c.Hooks.ApplyFilters("load_image_to_edit_attachmenturl", url, attachmentID, size); v != nil {
				switch t := v.(type) {
				case string:
					filepathStr = t
				case bool:
					if !t {
						filepathStr = ""
					}
				}
			} else {
				filepathStr = url
			}
		} else {
			filepathStr = url
		}
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("load_image_to_edit_path", filepathStr, attachmentID, size); v != nil {
			switch t := v.(type) {
			case string:
				return t
			case bool:
				if !t {
					return ""
				}
			}
		}
	}
	return filepathStr
}

// ImageGetIntermediateSize is image_get_intermediate_size subset (P8.28).
// Returns the sizes[size] bag (file/width/height) or nil.
func ImageGetIntermediateSize(c *Context, attachmentID int64, size string) map[string]any {
	size = strings.TrimSpace(size)
	if size == "" || size == "full" {
		return nil
	}
	meta, ok := GetAttachmentMetadata(c, attachmentID)
	if !ok || meta == nil {
		return nil
	}
	sizesRaw, ok := meta["sizes"]
	if !ok {
		return nil
	}
	var sizes map[string]any
	switch t := sizesRaw.(type) {
	case map[string]any:
		sizes = t
	case map[any]any:
		if m, ok := phpValueToStringMap(t); ok {
			sizes = m
		}
	}
	if sizes == nil {
		return nil
	}
	entry, ok := sizes[size]
	if !ok || entry == nil {
		return nil
	}
	switch t := entry.(type) {
	case map[string]any:
		return t
	case map[any]any:
		if m, ok := phpValueToStringMap(t); ok {
			return m
		}
	}
	return nil
}

// GetAttachmentBackupSizes reads _wp_attachment_backup_sizes (P8.28).
func GetAttachmentBackupSizes(c *Context, attachmentID int64) map[string]any {
	if c == nil || c.Meta == nil || attachmentID <= 0 {
		return nil
	}
	raw := c.Meta.GetMetadata(MetaPost, attachmentID, "_wp_attachment_backup_sizes", true)
	switch t := raw.(type) {
	case map[string]any:
		return t
	case map[any]any:
		if m, ok := phpValueToStringMap(t); ok {
			return m
		}
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		parsed, err := UnserializePHP(t)
		if err != nil {
			return nil
		}
		if m, ok := phpValueToStringMap(parsed); ok {
			return m
		}
	}
	return nil
}

// RecordImageEditBackupSizes stores a full-size backup entry before an edit (P8.28 stub).
// Mirrors the backup_sizes['full-orig'] bookkeeping in wp_save_image without pixel edits.
func RecordImageEditBackupSizes(c *Context, attachmentID int64) (map[string]any, error) {
	if c == nil || c.Meta == nil || attachmentID <= 0 {
		return nil, fmt.Errorf("wp: RecordImageEditBackupSizes: invalid args")
	}
	meta, ok := GetAttachmentMetadata(c, attachmentID)
	if !ok || meta == nil {
		return nil, fmt.Errorf("wp: Image data does not exist. Please re-upload the image.")
	}
	backup := GetAttachmentBackupSizes(c, attachmentID)
	if backup == nil {
		backup = map[string]any{}
	}
	if _, has := backup["full-orig"]; !has {
		file := strings.TrimSpace(fmt.Sprint(meta["file"]))
		if file == "" {
			file = filepath.Base(GetAttachedFile(c, attachmentID))
		}
		backup["full-orig"] = map[string]any{
			"file":   filepath.Base(file),
			"width":  meta["width"],
			"height": meta["height"],
		}
	}
	serialized, err := SerializePHP(backup)
	if err != nil {
		return nil, err
	}
	if _, _, _, err := c.Meta.UpdateMetadata(MetaPost, attachmentID, "_wp_attachment_backup_sizes", serialized, ""); err != nil {
		return nil, err
	}
	return backup, nil
}
