// is_serialized / maybe_serialize / maybe_unserialize (P2.123).
package wp

import (
	"regexp"
	"strings"
)

var (
	reSerializedStruct = regexp.MustCompile(`^(?:a|O|E):[0-9]+:`)
	reSerializedString = regexp.MustCompile(`^s:[0-9]+:`)
	reSerializedStrict = regexp.MustCompile(`^(?:b|i|d):[0-9.E+-]+;$`)
	reSerializedLoose  = regexp.MustCompile(`^(?:b|i|d):[0-9.E+-]+;`)
)

// IsSerialized mirrors is_serialized( $data, $strict = true ) (P2.123).
// Non-strings are never serialized. Enum token E is accepted (WP 6.1+).
func IsSerialized(data any, strict bool) bool {
	s, ok := data.(string)
	if !ok {
		return false
	}
	s = strings.TrimSpace(s)
	if s == "N;" {
		return true
	}
	if len(s) < 4 {
		return false
	}
	if s[1] != ':' {
		return false
	}
	if strict {
		last := s[len(s)-1]
		if last != ';' && last != '}' {
			return false
		}
	} else {
		semi := strings.IndexByte(s, ';')
		brace := strings.IndexByte(s, '}')
		if semi < 0 && brace < 0 {
			return false
		}
		if semi >= 0 && semi < 3 {
			return false
		}
		if brace >= 0 && brace < 4 {
			return false
		}
	}
	switch s[0] {
	case 's':
		if strict {
			if len(s) < 2 || s[len(s)-2] != '"' {
				return false
			}
		} else if !strings.Contains(s, `"`) {
			return false
		}
		return reSerializedString.MatchString(s)
	case 'a', 'O', 'E':
		return reSerializedStruct.MatchString(s)
	case 'b', 'i', 'd':
		if strict {
			return reSerializedStrict.MatchString(s)
		}
		return reSerializedLoose.MatchString(s)
	default:
		return false
	}
}

// IsSerializedString mirrors is_serialized_string() (P2.123).
func IsSerializedString(data any) bool {
	s, ok := data.(string)
	if !ok {
		return false
	}
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return false
	}
	if s[1] != ':' {
		return false
	}
	if !strings.HasSuffix(s, ";") {
		return false
	}
	if s[0] != 's' {
		return false
	}
	if s[len(s)-2] != '"' {
		return false
	}
	return true
}

// MaybeSerialize mirrors maybe_serialize() (P2.123).
// Arrays/maps/slices are PHP-serialized; already-serialized strings
// (loose check) are double-serialized for WP backward compatibility.
// Scalars are returned unchanged.
func MaybeSerialize(data any) any {
	switch data.(type) {
	case map[string]any, map[any]any, []any, []string, []int, []int64:
		s, err := SerializePHP(data)
		if err != nil {
			return data
		}
		return s
	}
	if s, ok := data.(string); ok {
		if IsSerialized(s, false) {
			out, err := SerializePHP(s)
			if err != nil {
				return data
			}
			return out
		}
		return s
	}
	return data
}

// MaybeUnserialize mirrors maybe_unserialize() (P2.123).
// Returns the original value when not a strict serialized string.
func MaybeUnserialize(data any) any {
	s, ok := data.(string)
	if !ok {
		return data
	}
	if !IsSerialized(s, true) {
		return data
	}
	v, err := UnserializePHP(strings.TrimSpace(s))
	if err != nil {
		return data
	}
	return v
}
