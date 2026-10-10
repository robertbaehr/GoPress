package wp

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// WPJSONEncode encodes value as JSON (wp_json_encode, P2.32).
//
// Mirrors the PHP helper: sanitize strings to valid UTF-8, then encode.
// Returns ("", false) when encoding fails. The optional flags argument is
// accepted for call-site parity with PHP but ignored; Go's encoding/json
// already HTML-escapes <>& similarly to common WP JSON_HEX_* flags.
func WPJSONEncode(value any, flags ...int) (string, bool) {
	_ = flags
	b, err := json.Marshal(sanitizeJSONValue(value, 512))
	if err != nil {
		return "", false
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

func sanitizeJSONValue(value any, depth int) any {
	if depth < 0 {
		return nil
	}
	switch v := value.(type) {
	case string:
		return sanitizeJSONString(v)
	case []any:
		out := make([]any, len(v))
		for i, el := range v {
			out[i] = sanitizeJSONValue(el, depth-1)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, el := range v {
			out[sanitizeJSONString(k)] = sanitizeJSONValue(el, depth-1)
		}
		return out
	default:
		return value
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
