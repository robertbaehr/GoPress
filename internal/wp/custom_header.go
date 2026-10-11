package wp

import (
	"fmt"
	"strings"
)

// GetHeaderImage returns theme_mod header_image, falling back to
// custom-header default-image support (get_header_image, P7.154).
// "remove-header" clears the image (WP sentinel).
func GetHeaderImage(c *Context) string {
	mods := loadThemeMods(c)
	if v, ok := mods["header_image"]; ok {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" || s == "remove-header" {
			return ""
		}
		return s
	}
	return customHeaderDefault(c, "default-image")
}

// GetHeaderTextColor returns theme_mod header_textcolor, falling back to
// custom-header default-text-color support (get_header_textcolor, P7.154).
func GetHeaderTextColor(c *Context) string {
	mods := loadThemeMods(c)
	if v, ok := mods["header_textcolor"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
			return strings.TrimPrefix(s, "#")
		}
	}
	if def := customHeaderDefault(c, "default-text-color"); def != "" {
		return strings.TrimPrefix(def, "#")
	}
	return ""
}

// SetHeaderImage persists theme_mod header_image URL (P7.154).
// Empty URL stores the WP "remove-header" sentinel.
func SetHeaderImage(c *Context, imageURL string) error {
	if c == nil || c.Options == nil {
		return fmt.Errorf("wp: SetHeaderImage: invalid state")
	}
	mods := loadThemeMods(c)
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		mods["header_image"] = "remove-header"
	} else {
		mods["header_image"] = imageURL
	}
	return saveThemeMods(c, mods)
}

// SetHeaderTextColor persists theme_mod header_textcolor (hex without #, P7.154).
// Empty color removes the mod.
func SetHeaderTextColor(c *Context, color string) error {
	if c == nil || c.Options == nil {
		return fmt.Errorf("wp: SetHeaderTextColor: invalid state")
	}
	mods := loadThemeMods(c)
	color = strings.TrimPrefix(strings.TrimSpace(color), "#")
	if color == "" {
		delete(mods, "header_textcolor")
	} else {
		mods["header_textcolor"] = color
	}
	return saveThemeMods(c, mods)
}

// HasHeaderImage reports whether a custom header image is set (has_header_image, P7.154).
func HasHeaderImage(c *Context) bool {
	return GetHeaderImage(c) != ""
}

// HasCustomHeader reports whether the front should emit the custom-header
// body class (theme supports + header image or non-default text color, P7.154).
func HasCustomHeader(c *Context) bool {
	if !CurrentThemeSupports(c, "custom-header") {
		return false
	}
	if HasHeaderImage(c) {
		return true
	}
	color := GetHeaderTextColor(c)
	if color == "" {
		return false
	}
	def := strings.TrimPrefix(customHeaderDefault(c, "default-text-color"), "#")
	return !strings.EqualFold(color, def)
}

func customHeaderDefault(c *Context, key string) string {
	if c == nil || !CurrentThemeSupports(c, "custom-header") {
		return ""
	}
	raw := GetThemeSupport(c, "custom-header")
	m := themeSupportMap(raw)
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" {
		return ""
	}
	return s
}
