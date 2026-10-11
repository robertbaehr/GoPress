package wp

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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

// ParseList splits a comma- or whitespace-separated list (wp_parse_list, P2.91 / P2.122).
// Integers become a single-element list; empty / whitespace-only string → empty slice.
// Array inputs keep empty-string scalars (PHP array_filter is_scalar) and stringify
// numeric/bool entries; nil / non-scalar array entries are dropped.
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
		// P2.122: PHP casts floats via (string) before preg_split ("1.5", not "1").
		return parseListFromString(phpStringCastFloat(v))
	case float32:
		return parseListFromString(phpStringCastFloat(float64(v)))
	case bool:
		if v {
			return []string{"1"}
		}
		return []string{}
	case string:
		return parseListFromString(v)
	case []string:
		// P2.122: preserve empty strings like PHP array_filter(..., 'is_scalar').
		out := make([]string, 0, len(v))
		out = append(out, v...)
		return out
	case []int:
		out := make([]string, len(v))
		for i, n := range v {
			out[i] = strconv.Itoa(n)
		}
		return out
	case []int64:
		out := make([]string, len(v))
		for i, n := range v {
			out[i] = strconv.FormatInt(n, 10)
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if item == nil {
				continue
			}
			switch s := item.(type) {
			case string:
				out = append(out, s)
			case bool:
				if s {
					out = append(out, "1")
				} else {
					out = append(out, "")
				}
			case int:
				out = append(out, strconv.Itoa(s))
			case int8:
				out = append(out, strconv.Itoa(int(s)))
			case int16:
				out = append(out, strconv.Itoa(int(s)))
			case int32:
				out = append(out, strconv.Itoa(int(s)))
			case int64:
				out = append(out, strconv.FormatInt(s, 10))
			case uint, uint8, uint16, uint32, uint64:
				out = append(out, fmt.Sprint(s))
			case float64:
				out = append(out, phpStringCastFloat(s))
			case float32:
				out = append(out, phpStringCastFloat(float64(s)))
			default:
				// Non-scalar (maps/slices/structs) dropped like !is_scalar.
			}
		}
		return out
	default:
		return nil
	}
}

func parseListFromString(v string) []string {
	parts := parseListSplit.Split(v, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// phpStringCastFloat mirrors PHP (string)$float for wp_parse_list scalars (P2.122).
func phpStringCastFloat(f float64) string {
	// Prefer integer form when the float is integral (PHP: (string)3.0 === "3").
	if f == float64(int64(f)) && !isNegZero(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'G', -1, 64)
}

func isNegZero(f float64) bool {
	return f == 0 && strconv.FormatFloat(f, 'f', -1, 64) == "-0"
}

// ParseIDList cleans a list of IDs (wp_parse_id_list, P2.122).
// Returns unique abs(int) values; may include 0. Order follows first occurrence.
func ParseIDList(input any) []int64 {
	list := ParseList(input)
	seen := map[int64]bool{}
	out := make([]int64, 0, len(list))
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			// absint('') → 0 in PHP.
			if !seen[0] {
				seen[0] = true
				out = append(out, 0)
			}
			continue
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			// Non-numeric → absint → 0.
			if !seen[0] {
				seen[0] = true
				out = append(out, 0)
			}
			continue
		}
		if n < 0 {
			n = -n
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// ParseSlugList cleans a list of slugs (wp_parse_slug_list, P2.122).
// Runs sanitize_title on each entry and unique's the result.
func ParseSlugList(input any) []string {
	list := ParseList(input)
	seen := map[string]bool{}
	out := make([]string, 0, len(list))
	for _, s := range list {
		slug := SanitizeTitle(fmt.Sprint(s))
		if !seen[slug] {
			seen[slug] = true
			out = append(out, slug)
		}
	}
	return out
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
