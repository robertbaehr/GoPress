// wp_slash / wp_unslash (P2.33 / P2.85 / P2.124) — subset of wp-includes/formatting.php.
package wp

// MapDeepCallback is applied to non-container leaves by MapDeep (map_deep).
type MapDeepCallback func(value any) any

// MapDeep recursively walks slices/maps and applies fn to leaves (map_deep, P2.124).
// Container types mirror PHP arrays: []any, []string, map[string]any,
// map[string]string, map[any]any, map[string][]string, []map[string]any,
// map[string][]any.
func MapDeep(value any, fn MapDeepCallback) any {
	if fn == nil {
		return value
	}
	switch v := value.(type) {
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = MapDeep(item, fn)
		}
		return out
	case []string:
		out := make([]string, len(v))
		for i, item := range v {
			if s, ok := MapDeep(item, fn).(string); ok {
				out[i] = s
			} else {
				out[i] = item
			}
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(v))
		for i, item := range v {
			if m, ok := MapDeep(item, fn).(map[string]any); ok {
				out[i] = m
			} else {
				out[i] = item
			}
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = MapDeep(item, fn)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, item := range v {
			if s, ok := MapDeep(item, fn).(string); ok {
				out[k] = s
			} else {
				out[k] = item
			}
		}
		return out
	case map[string][]string:
		out := make(map[string][]string, len(v))
		for k, item := range v {
			if s, ok := MapDeep(item, fn).([]string); ok {
				out[k] = s
			} else {
				out[k] = item
			}
		}
		return out
	case map[string][]any:
		out := make(map[string][]any, len(v))
		for k, item := range v {
			if s, ok := MapDeep(item, fn).([]any); ok {
				out[k] = s
			} else {
				out[k] = item
			}
		}
		return out
	case map[any]any:
		out := make(map[any]any, len(v))
		for k, item := range v {
			out[k] = MapDeep(item, fn)
		}
		return out
	default:
		return fn(value)
	}
}

// Slash adds slashes to a string, or recursively to strings in a slice/map
// (wp_slash). Non-string values are returned unchanged.
//
// P2.85: also walks map[string]string and map[any]any (PHP array_map depth).
// P2.124: MapDeep backbone + map[string][]string / []map[string]any /
// map[string][]any.
func Slash(value any) any {
	return MapDeep(value, func(v any) any {
		if s, ok := v.(string); ok {
			return phpAddslashes(s)
		}
		return v
	})
}

// SlashString is wp_slash for a single string.
func SlashString(s string) string {
	return phpAddslashes(s)
}

// Unslash removes slashes from a string, or recursively from strings in a
// slice/map (wp_unslash / stripslashes_deep).
//
// P2.85: also walks map[string]string and map[any]any.
// P2.124: uses MapDeep + stripslashes_from_strings_only (WP wp_unslash).
func Unslash(value any) any {
	return MapDeep(value, StripslashesFromStringsOnly)
}

// UnslashString is wp_unslash for a single string.
func UnslashString(s string) string {
	return phpStripslashes(s)
}

// StripslashesDeep is stripslashes_deep — alias of Unslash for PHP parity (P2.85).
func StripslashesDeep(value any) any {
	return Unslash(value)
}

// StripslashesFromStringsOnly is stripslashes_from_strings_only (P2.85).
// Non-strings pass through unchanged (unlike Unslash which only walks containers).
func StripslashesFromStringsOnly(value any) any {
	if s, ok := value.(string); ok {
		return phpStripslashes(s)
	}
	return value
}
