// wp_slash / wp_unslash (P2.33 / P2.85) — subset of wp-includes/formatting.php.
package wp

// Slash adds slashes to a string, or recursively to strings in a slice/map
// (wp_slash). Non-string values are returned unchanged.
//
// P2.85: also walks map[string]string and map[any]any (PHP array_map depth).
func Slash(value any) any {
	switch v := value.(type) {
	case string:
		return phpAddslashes(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = Slash(item)
		}
		return out
	case []string:
		out := make([]string, len(v))
		for i, item := range v {
			out[i] = phpAddslashes(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = Slash(item)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, item := range v {
			out[k] = phpAddslashes(item)
		}
		return out
	case map[any]any:
		out := make(map[any]any, len(v))
		for k, item := range v {
			out[k] = Slash(item)
		}
		return out
	default:
		return value
	}
}

// SlashString is wp_slash for a single string.
func SlashString(s string) string {
	return phpAddslashes(s)
}

// Unslash removes slashes from a string, or recursively from strings in a
// slice/map (wp_unslash / stripslashes_deep).
//
// P2.85: also walks map[string]string and map[any]any.
func Unslash(value any) any {
	switch v := value.(type) {
	case string:
		return phpStripslashes(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = Unslash(item)
		}
		return out
	case []string:
		out := make([]string, len(v))
		for i, item := range v {
			out[i] = phpStripslashes(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = Unslash(item)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, item := range v {
			out[k] = phpStripslashes(item)
		}
		return out
	case map[any]any:
		out := make(map[any]any, len(v))
		for k, item := range v {
			out[k] = Unslash(item)
		}
		return out
	default:
		return value
	}
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
