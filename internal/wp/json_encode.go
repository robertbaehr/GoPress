package wp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// Common PHP json_encode flag bits honored by WPJSONEncode (P2.121).
const (
	JSONPrettyPrint    = 128 // JSON_PRETTY_PRINT
	JSONUnescapedSlash = 64  // JSON_UNESCAPED_SLASHES (Go encoding/json always escapes /? no — Go does not escape /)
)

// WPJSONEncode encodes value as JSON (wp_json_encode, P2.32 / P2.121).
//
// Mirrors the PHP helper: sanitize strings to valid UTF-8 (and nested maps/slices),
// then encode. Returns ("", false) when encoding fails or the depth limit is hit.
//
// Optional trailing ints: flags (bitmask), depth (default 512).
// JSON_PRETTY_PRINT (128) is honored; other flags are accepted for call-site parity.
func WPJSONEncode(value any, flagsAndDepth ...int) (string, bool) {
	flags := 0
	depth := 512
	if len(flagsAndDepth) > 0 {
		flags = flagsAndDepth[0]
	}
	if len(flagsAndDepth) > 1 {
		depth = flagsAndDepth[1]
	}
	if depth <= 0 {
		return "", false
	}
	sanitized, ok := sanitizeJSONValue(value, depth)
	if !ok {
		return "", false
	}
	var b []byte
	var err error
	if flags&JSONPrettyPrint != 0 {
		b, err = json.MarshalIndent(sanitized, "", "    ")
	} else {
		b, err = json.Marshal(sanitized)
	}
	if err != nil {
		return "", false
	}
	// PHP JSON_UNESCAPED_SLASHES: undo Go's default \/ escape if any (Go rarely emits \/).
	if flags&JSONUnescapedSlash != 0 {
		b = bytes.ReplaceAll(b, []byte(`\/`), []byte(`/`))
	}
	return string(b), true
}

// WPJSONEncodeStrict is like WPJSONEncode but returns an error instead of bool.
func WPJSONEncodeStrict(value any) (string, error) {
	s, ok := WPJSONEncode(value)
	if !ok {
		return "", fmt.Errorf("wp_json_encode failed")
	}
	return s, nil
}

// sanitizeJSONValue walks value like _wp_json_sanity_check (P2.121).
// ok=false when depth is exhausted (depth < 0 at entry of a nested container).
func sanitizeJSONValue(value any, depth int) (any, bool) {
	if depth < 0 {
		return nil, false
	}
	switch v := value.(type) {
	case nil:
		return nil, true
	case string:
		return sanitizeJSONString(v), true
	case []byte:
		return sanitizeJSONString(string(v)), true
	case []any:
		out := make([]any, len(v))
		for i, el := range v {
			s, ok := sanitizeJSONValue(el, depth-1)
			if !ok {
				return nil, false
			}
			out[i] = s
		}
		return out, true
	case []string:
		out := make([]any, len(v))
		for i, el := range v {
			out[i] = sanitizeJSONString(el)
		}
		return out, true
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, el := range v {
			s, ok := sanitizeJSONValue(el, depth-1)
			if !ok {
				return nil, false
			}
			out[sanitizeJSONString(k)] = s
		}
		return out, true
	case map[string]string:
		out := make(map[string]any, len(v))
		for k, el := range v {
			out[sanitizeJSONString(k)] = sanitizeJSONString(el)
		}
		return out, true
	case map[any]any:
		out := make(map[string]any, len(v))
		for k, el := range v {
			ks := sanitizeJSONString(fmt.Sprint(k))
			s, ok := sanitizeJSONValue(el, depth-1)
			if !ok {
				return nil, false
			}
			out[ks] = s
		}
		return out, true
	default:
		return value, true
	}
}

func sanitizeJSONString(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return stringsToValidUTF8(s)
}

func stringsToValidUTF8(s string) string {
	var b []byte
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			b = append(b, []byte(string(utf8.RuneError))...)
			s = s[1:]
			continue
		}
		b = append(b, s[:size]...)
		s = s[size:]
	}
	return string(b)
}
