package wp

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseArgs merges args into defaults (wp_parse_args, P2.30).
//
// args may be:
//   - map[string]any / map[string]string
//   - url.Values
//   - a query string (a=1&b=2), including PHP-style foo[bar]=baz keys as flat keys
//   - nil / other → treated as empty
//
// Defaults are shallow-copied; args keys overwrite. Returns a new map.
func ParseArgs(args any, defaults map[string]any) map[string]any {
	parsed := coerceParseArgs(args)
	if len(defaults) == 0 {
		if parsed == nil {
			return map[string]any{}
		}
		return parsed
	}
	out := make(map[string]any, len(defaults)+len(parsed))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range parsed {
		out[k] = v
	}
	return out
}

func coerceParseArgs(args any) map[string]any {
	switch t := args.(type) {
	case nil:
		return nil
	case map[string]any:
		if t == nil {
			return nil
		}
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[k] = v
		}
		return out
	case map[string]string:
		if t == nil {
			return nil
		}
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[k] = v
		}
		return out
	case url.Values:
		return valuesToAnyMap(t)
	case string:
		return parseArgsQuery(t)
	case []byte:
		return parseArgsQuery(string(t))
	default:
		// Best-effort: fmt.Sprint for scalars is not useful; return empty.
		_ = fmt.Sprintf("%T", t)
		return nil
	}
}

func parseArgsQuery(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	// Strip leading ? if present.
	raw = strings.TrimPrefix(raw, "?")
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return map[string]any{}
	}
	return valuesToAnyMap(vals)
}

func valuesToAnyMap(vals url.Values) map[string]any {
	out := make(map[string]any, len(vals))
	for k, vs := range vals {
		if len(vs) == 0 {
			out[k] = ""
			continue
		}
		if len(vs) == 1 {
			out[k] = vs[0]
			continue
		}
		cp := make([]any, len(vs))
		for i, v := range vs {
			cp[i] = v
		}
		out[k] = cp
	}
	return out
}
