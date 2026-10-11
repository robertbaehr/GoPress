package wp

import (
	"path/filepath"
	"strconv"
	"strings"
)

// MimeTypeIcon returns a media mime icon URL (wp_mime_type_icon subset, P8.53).
// mimeOrID may be a post ID (numeric string), mime type, or extension.
// preferredExt defaults to ".png"; ".svg" also accepted.
// Applies icon_dir_uri then wp_mime_type_icon filters.
func MimeTypeIcon(c *Context, mimeOrID string, preferredExt string) string {
	preferredExt = strings.TrimSpace(preferredExt)
	if preferredExt == "" {
		preferredExt = ".png"
	}
	if !strings.HasPrefix(preferredExt, ".") {
		preferredExt = "." + strings.ToLower(preferredExt)
	} else {
		preferredExt = strings.ToLower(preferredExt)
	}
	if preferredExt != ".png" && preferredExt != ".svg" {
		preferredExt = ".png"
	}

	types := []string{}
	mime := strings.TrimSpace(mimeOrID)
	if mime == "" {
		return ""
	}
	if id, err := strconv.ParseInt(mime, 10, 64); err == nil && id > 0 {
		file := GetAttachedFile(c, id)
		if file != "" {
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
			if ext != "" {
				types = append(types, ext)
				if t := Ext2Type(c, ext); t != "" {
					types = append(types, t)
				}
			}
		}
		if m, ok := GetPostMimeType(c, id); ok && strings.TrimSpace(m) != "" {
			mime = m
		} else {
			mime = ""
		}
	}
	if mime != "" {
		types = append(types, mime)
		if i := strings.IndexByte(mime, '/'); i > 0 {
			types = append(types, mime[:i])
		}
	}

	baseURI := includesMediaIconURI(c)
	icon := baseURI + "/default" + preferredExt
	known := map[string]bool{
		"archive": true, "audio": true, "code": true, "document": true,
		"interactive": true, "spreadsheet": true, "text": true, "video": true,
		"default": true,
	}
	for _, candidate := range types {
		cand := strings.ToLower(strings.TrimSpace(candidate))
		if cand == "" {
			continue
		}
		// Map mime major / ext2type names onto icon basenames.
		name := cand
		if t := Ext2Type(c, cand); t != "" {
			name = t
		}
		switch {
		case known[name]:
			icon = baseURI + "/" + name + preferredExt
			goto done
		case strings.HasPrefix(cand, "audio"):
			icon = baseURI + "/audio" + preferredExt
			goto done
		case strings.HasPrefix(cand, "video"):
			icon = baseURI + "/video" + preferredExt
			goto done
		case strings.HasPrefix(cand, "text"):
			icon = baseURI + "/text" + preferredExt
			goto done
		case strings.Contains(cand, "zip") || strings.Contains(cand, "archive"):
			icon = baseURI + "/archive" + preferredExt
			goto done
		}
	}
done:
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_mime_type_icon", icon, mimeOrID, preferredExt); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return icon
}

func includesMediaIconURI(c *Context) string {
	base := SiteURL(c, "wp-includes/images/media", "https")
	base = strings.TrimRight(base, "/")
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("icon_dir_uri", base); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimRight(strings.TrimSpace(s), "/")
			}
		}
	}
	return base
}
