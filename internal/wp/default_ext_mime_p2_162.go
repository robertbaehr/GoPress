package wp

import (
	"sort"
	"strings"
)

// GetDefaultExtensionForMimeType returns the first file extension for a mime
// type from GetMimeTypes (wp_get_default_extension_for_mime_type, P2.162).
// Uses PHP mime map insertion order (array_search parity). Returns "" when
// the mime is unknown (WP returns false).
func GetDefaultExtensionForMimeType(c *Context, mimeType string) string {
	mimeType = strings.TrimSpace(mimeType)
	if mimeType == "" {
		return ""
	}
	mimes := GetMimeTypes(c)
	// Walk default order first (array_search on the PHP ordered array).
	for _, key := range defaultMimeTypeOrder {
		if mimes[key] != mimeType {
			continue
		}
		parts := strings.Split(key, "|")
		if len(parts) == 0 {
			continue
		}
		ext := strings.TrimSpace(parts[0])
		if ext != "" {
			return ext
		}
	}
	// Filter-added keys (not in default order): stable sort then scan.
	var extras []string
	for key := range mimes {
		if _, ok := defaultMimeTypes[key]; ok {
			continue
		}
		extras = append(extras, key)
	}
	sort.Strings(extras)
	for _, key := range extras {
		if mimes[key] != mimeType {
			continue
		}
		parts := strings.Split(key, "|")
		if len(parts) == 0 {
			continue
		}
		ext := strings.TrimSpace(parts[0])
		if ext != "" {
			return ext
		}
	}
	return ""
}
