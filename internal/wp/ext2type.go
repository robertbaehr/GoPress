// wp_ext2type / wp_get_ext_types (P2.55) — subset of wp-includes/functions.php.
package wp

import "strings"

// defaultExtTypes mirrors wp_get_ext_types() defaults before the ext2type filter.
var defaultExtTypes = map[string][]string{
	"image":       {"jpg", "jpeg", "jpe", "gif", "png", "bmp", "tif", "tiff", "ico", "heic", "heif", "webp", "avif"},
	"audio":       {"aac", "ac3", "aif", "aiff", "flac", "m3a", "m4a", "m4b", "mka", "mp1", "mp2", "mp3", "ogg", "oga", "ram", "wav", "wma"},
	"video":       {"3g2", "3gp", "3gpp", "asf", "avi", "divx", "dv", "flv", "m4v", "mkv", "mov", "mp4", "mpeg", "mpg", "mpv", "ogm", "ogv", "qt", "rm", "vob", "wmv"},
	"document":    {"doc", "docx", "docm", "dotm", "odt", "pages", "pdf", "xps", "oxps", "rtf", "wp", "wpd", "psd", "xcf"},
	"spreadsheet": {"numbers", "ods", "xls", "xlsx", "xlsm", "xlsb"},
	"interactive": {"swf", "key", "ppt", "pptx", "pptm", "pps", "ppsx", "ppsm", "sldx", "sldm", "odp"},
	"text":        {"asc", "csv", "tsv", "txt"},
	"archive":     {"bz2", "cab", "dmg", "gz", "rar", "sea", "sit", "sqx", "tar", "tgz", "zip", "7z"},
	"code":        {"css", "htm", "html", "php", "js"},
}

// GetExtTypes returns the extension→type map (wp_get_ext_types).
// When c is non-nil, applies the "ext2type" filter.
func GetExtTypes(c *Context) map[string][]string {
	out := copyExtTypes(defaultExtTypes)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("ext2type", out); v != nil {
			if m, ok := v.(map[string][]string); ok {
				return m
			}
		}
	}
	return out
}

// Ext2Type returns the file type for an extension (wp_ext2type), or "".
// Example: Ext2Type(nil, "mp3") → "audio".
func Ext2Type(c *Context, ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		return ""
	}
	for typ, exts := range GetExtTypes(c) {
		for _, e := range exts {
			if e == ext {
				return typ
			}
		}
	}
	return ""
}

func copyExtTypes(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for k, v := range in {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}
