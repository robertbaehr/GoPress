package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// defaultRegisteredNavMenus are classic menu locations GoPress exposes (P4.41).
// Block themes rarely register these; GoPress keeps a small stable set for REST.
var defaultRegisteredNavMenus = map[string]string{
	"primary": "Primary Menu",
	"footer":  "Footer Menu",
}

// RegisteredNavMenus returns registered nav menu location name → description.
func RegisteredNavMenus(c *Context) map[string]string {
	_ = c
	out := make(map[string]string, len(defaultRegisteredNavMenus))
	for k, v := range defaultRegisteredNavMenus {
		out[k] = v
	}
	return out
}

// GetNavMenuLocations returns location → menu term ID assignments (theme_mod).
func GetNavMenuLocations(c *Context) map[string]int64 {
	out := map[string]int64{}
	mods := loadThemeMods(c)
	raw, ok := mods["nav_menu_locations"]
	if !ok || raw == nil {
		return out
	}
	switch t := raw.(type) {
	case map[string]any:
		for loc, v := range t {
			out[loc] = anyToInt64(v)
		}
	case map[any]any:
		for k, v := range t {
			out[fmt.Sprint(k)] = anyToInt64(v)
		}
	}
	return out
}

// SetNavMenuLocation assigns menuID to location (0 clears). Persists theme_mod (P4.41).
func SetNavMenuLocation(c *Context, location string, menuID int64) error {
	location = strings.TrimSpace(location)
	if c == nil || c.Options == nil || location == "" {
		return fmt.Errorf("wp: SetNavMenuLocation: invalid state")
	}
	if _, ok := RegisteredNavMenus(c)[location]; !ok {
		return fmt.Errorf("wp: invalid menu location %q", location)
	}
	mods := loadThemeMods(c)
	locs := map[string]any{}
	if existing, ok := mods["nav_menu_locations"].(map[string]any); ok {
		for k, v := range existing {
			locs[k] = v
		}
	} else if existing, ok := mods["nav_menu_locations"].(map[any]any); ok {
		for k, v := range existing {
			locs[fmt.Sprint(k)] = v
		}
	}
	if menuID <= 0 {
		delete(locs, location)
	} else {
		locs[location] = menuID
	}
	mods["nav_menu_locations"] = locs
	return saveThemeMods(c, mods)
}

func themeModsOptionName(c *Context) string {
	return "theme_mods_" + ActiveStylesheet(c)
}

func loadThemeMods(c *Context) map[string]any {
	out := map[string]any{}
	if c == nil || c.Options == nil {
		return out
	}
	raw := c.Options.GetOption(themeModsOptionName(c), "")
	s, _ := raw.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return out
	}
	v, err := phpUnserialize(s)
	if err != nil {
		return out
	}
	switch t := v.(type) {
	case map[string]any:
		return t
	case map[any]any:
		for k, val := range t {
			out[fmt.Sprint(k)] = val
		}
	}
	return out
}

func saveThemeMods(c *Context, mods map[string]any) error {
	if c == nil || c.Options == nil {
		return fmt.Errorf("wp: saveThemeMods: invalid state")
	}
	if mods == nil {
		mods = map[string]any{}
	}
	serialized, err := phpSerialize(mods)
	if err != nil {
		return err
	}
	_, err = c.Options.UpdateOption(themeModsOptionName(c), serialized, "yes")
	return err
}

func anyToInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
		return n
	}
}
