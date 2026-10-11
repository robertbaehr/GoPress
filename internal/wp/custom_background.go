package wp

import (
	"fmt"
	"strings"
)

// GetBackgroundColor returns theme_mod background_color, falling back to
// custom-background default-color support (get_background_color, P7.153).
func GetBackgroundColor(c *Context) string {
	mods := loadThemeMods(c)
	if v, ok := mods["background_color"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
			return strings.TrimPrefix(s, "#")
		}
	}
	if def := customBackgroundDefault(c, "default-color"); def != "" {
		return strings.TrimPrefix(def, "#")
	}
	return ""
}

// GetBackgroundImage returns theme_mod background_image, falling back to
// custom-background default-image support (get_background_image, P7.153).
func GetBackgroundImage(c *Context) string {
	mods := loadThemeMods(c)
	if v, ok := mods["background_image"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
			return s
		}
	}
	return customBackgroundDefault(c, "default-image")
}

// SetBackgroundColor persists theme_mod background_color (hex without #, P7.153).
// Empty color removes the mod.
func SetBackgroundColor(c *Context, color string) error {
	if c == nil || c.Options == nil {
		return fmt.Errorf("wp: SetBackgroundColor: invalid state")
	}
	mods := loadThemeMods(c)
	color = strings.TrimPrefix(strings.TrimSpace(color), "#")
	if color == "" {
		delete(mods, "background_color")
	} else {
		mods["background_color"] = color
	}
	return saveThemeMods(c, mods)
}

// SetBackgroundImage persists theme_mod background_image URL (P7.153).
// Empty URL removes the mod.
func SetBackgroundImage(c *Context, imageURL string) error {
	if c == nil || c.Options == nil {
		return fmt.Errorf("wp: SetBackgroundImage: invalid state")
	}
	mods := loadThemeMods(c)
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		delete(mods, "background_image")
	} else {
		mods["background_image"] = imageURL
	}
	return saveThemeMods(c, mods)
}

// HasCustomBackground reports whether the front should emit the custom-background
// body class (theme supports + non-default color or image, P7.153).
func HasCustomBackground(c *Context) bool {
	if !CurrentThemeSupports(c, "custom-background") {
		return false
	}
	img := GetBackgroundImage(c)
	if img != "" {
		return true
	}
	color := GetBackgroundColor(c)
	if color == "" {
		return false
	}
	def := strings.TrimPrefix(customBackgroundDefault(c, "default-color"), "#")
	return !strings.EqualFold(color, def)
}

func customBackgroundDefault(c *Context, key string) string {
	if c == nil || !CurrentThemeSupports(c, "custom-background") {
		return ""
	}
	raw := GetThemeSupport(c, "custom-background")
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
