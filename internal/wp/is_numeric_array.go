package wp

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

var parseListSplit = regexp.MustCompile(`[\s,]+`)

// IsNumericArray reports whether data is a list-like array whose keys are all
// non-string (wp_is_numeric_array, P2.36 / P2.91).
//
// In Go: slices/arrays are numeric; maps are numeric only when every key is an
// integer kind (empty maps count as numeric). String keys — including digit
// strings — make the map non-numeric (PHP: ! is_string( $key )). Non-array
// values return false.
func IsNumericArray(data any) bool {
	if data == nil {
		return false
	}
	rv := reflect.ValueOf(data)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return true
	case reflect.Map:
		for _, k := range rv.MapKeys() {
			if !isNumericArrayKey(k) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func isNumericArrayKey(k reflect.Value) bool {
	for k.Kind() == reflect.Interface {
		if k.IsNil() {
			return false
		}
		k = k.Elem()
	}
	switch k.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		// PHP rejects string keys (even digit strings typed as string).
		return false
	}
}

// ParseList splits a comma- or whitespace-separated list (wp_parse_list, P2.91).
// Integers become a single-element list; empty / whitespace-only → empty slice.
func ParseList(input any) []string {
	switch v := input.(type) {
	case nil:
		return nil
	case int:
		return []string{strconv.Itoa(v)}
	case int8:
		return []string{strconv.Itoa(int(v))}
	case int16:
		return []string{strconv.Itoa(int(v))}
	case int32:
		return []string{strconv.Itoa(int(v))}
	case int64:
		return []string{strconv.FormatInt(v, 10)}
	case uint, uint8, uint16, uint32, uint64:
		return []string{fmt.Sprint(v)}
	case float64:
		return []string{strconv.FormatInt(int64(v), 10)}
	case float32:
		return []string{strconv.FormatInt(int64(v), 10)}
	case bool:
		if v {
			return []string{"1"}
		}
		return []string{}
	case string:
		parts := parseListSplit.Split(v, -1)
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			switch s := item.(type) {
			case string:
				if s != "" {
					out = append(out, s)
				}
			case int, int64, float64, bool:
				out = append(out, ParseList(s)...)
			}
		}
		return out
	default:
		return nil
	}
}

// RestIsArray reports whether value is array-like (rest_is_array, P2.91).
// Scalars are first expanded via ParseList (wp_parse_list).
func RestIsArray(maybe any) bool {
	if maybe == nil {
		return false
	}
	switch maybe.(type) {
	case string, bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, float32, float64:
		return IsNumericArray(ParseList(maybe))
	default:
		return IsNumericArray(maybe)
	}
}

// RestSanitizeArray converts an array-like value to a numeric list
// (rest_sanitize_array, P2.91). Scalars go through ParseList; non-arrays → empty.
func RestSanitizeArray(maybe any) []any {
	if maybe == nil {
		return []any{}
	}
	switch maybe.(type) {
	case string, bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, float32, float64:
		list := ParseList(maybe)
		out := make([]any, len(list))
		for i, s := range list {
			out[i] = s
		}
		return out
	}
	rv := reflect.ValueOf(maybe)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return []any{}
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out
	case reflect.Map:
		// Normalize to numeric array so nothing unexpected is in the keys.
		out := make([]any, 0, rv.Len())
		for _, k := range rv.MapKeys() {
			out = append(out, rv.MapIndex(k).Interface())
		}
		return out
	default:
		return []any{}
	}
}
