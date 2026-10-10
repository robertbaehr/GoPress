// wp_slash / wp_unslash (P2.33) — subset of wp-includes/formatting.php.
package wp

// Slash adds slashes to a string, or recursively to strings in a slice/map
// (wp_slash). Non-string values are returned unchanged.
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
	default:
		return value
	}
}

// UnslashString is wp_unslash for a single string.
func UnslashString(s string) string {
	return phpStripslashes(s)
}
