package wp

import (
	"path/filepath"
	"strings"
)

// GetOriginalImagePath returns the filesystem path to the originally uploaded
// image (wp_get_original_image_path, P8.54). Non-images → "". Applies
// wp_get_original_image_path filter.
func GetOriginalImagePath(c *Context, attachmentID int64) string {
	// Match wp_attachment_is_image → wp_attachment_is('image') (no preview-src fallback).
	if c == nil || attachmentID <= 0 || !AttachmentIs(c, "image", attachmentID) {
		return ""
	}
	imageFile := GetAttachedFile(c, attachmentID)
	out := imageFile
	if meta, ok := GetAttachmentMetadata(c, attachmentID); ok {
		if orig, _ := meta["original_image"].(string); strings.TrimSpace(orig) != "" {
			out = filepath.Join(filepath.Dir(imageFile), filepath.Base(strings.TrimSpace(orig)))
		}
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_get_original_image_path", out, attachmentID); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return out
}

// GetOriginalImageURL returns the URL of the originally uploaded image
// (wp_get_original_image_url, P8.54). Non-images / empty URL → "".
// Applies wp_get_original_image_url filter.
func GetOriginalImageURL(c *Context, attachmentID int64) string {
	if c == nil || attachmentID <= 0 || !AttachmentIs(c, "image", attachmentID) {
		return ""
	}
	imageURL := GetAttachmentURL(c, attachmentID)
	if imageURL == "" {
		return ""
	}
	out := imageURL
	if meta, ok := GetAttachmentMetadata(c, attachmentID); ok {
		if orig, _ := meta["original_image"].(string); strings.TrimSpace(orig) != "" {
			base := strings.TrimSpace(orig)
			base = filepath.Base(base)
			dir := imageURL
			if i := strings.LastIndex(imageURL, "/"); i >= 0 {
				dir = imageURL[:i]
			}
			out = strings.TrimRight(dir, "/") + "/" + base
		}
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_get_original_image_url", out, attachmentID); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return out
}
