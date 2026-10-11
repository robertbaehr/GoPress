// wp_validate_boolean / rest_is_boolean / rest_sanitize_boolean (P2.34 / P2.120).
package wp

import (
	"reflect"
	"strings"
)

// ValidateBoolean mirrors wp_validate_boolean: bools pass through; the string
// "false" (any case) is false; otherwise PHP-like truthiness applies.
func ValidateBoolean(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		if strings.EqualFold(v, "false") {
			return false
		}
		return phpTruthyString(v)
	case []byte:
		return ValidateBoolean(string(v))
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	case complex64:
		return v != 0
	case complex128:
		return v != 0
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Map, reflect.Array:
			return rv.Len() > 0
		case reflect.Ptr, reflect.Interface:
			if rv.IsNil() {
				return false
			}
			return ValidateBoolean(rv.Elem().Interface())
		case reflect.String:
			return ValidateBoolean(rv.String())
		case reflect.Bool:
			return rv.Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return rv.Int() != 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return rv.Uint() != 0
		case reflect.Float32, reflect.Float64:
			return rv.Float() != 0
		default:
			// Objects / structs / funcs: PHP (bool) object is true.
			return value != nil
		}
	}
}

// RestIsBoolean mirrors rest_is_boolean: bool, "true"/"false"/"0"/"1", or int 0/1 (P2.120).
func RestIsBoolean(value any) bool {
	switch v := value.(type) {
	case bool:
		return true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "false", "true", "0", "1":
			return true
		default:
			return false
		}
	case []byte:
		return RestIsBoolean(string(v))
	case int:
		return v == 0 || v == 1
	case int8:
		return v == 0 || v == 1
	case int16:
		return v == 0 || v == 1
	case int32:
		return v == 0 || v == 1
	case int64:
		return v == 0 || v == 1
	case uint:
		return v == 0 || v == 1
	case uint8:
		return v == 0 || v == 1
	case uint16:
		return v == 0 || v == 1
	case uint32:
		return v == 0 || v == 1
	case uint64:
		return v == 0 || v == 1
	case float64:
		// JSON numbers often arrive as float64.
		return v == 0 || v == 1
	case float32:
		return v == 0 || v == 1
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Bool:
			return true
		case reflect.String:
			return RestIsBoolean(rv.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := rv.Int()
			return n == 0 || n == 1
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n := rv.Uint()
			return n == 0 || n == 1
		case reflect.Float32, reflect.Float64:
			n := rv.Float()
			return n == 0 || n == 1
		default:
			return false
		}
	}
}

// RestSanitizeBoolean mirrors rest_sanitize_boolean (P2.120).
// Strings "false"/"0" (any case) become false; otherwise PHP (bool) casting applies
// via ValidateBoolean (after the string false/0 special case).
func RestSanitizeBoolean(value any) bool {
	switch v := value.(type) {
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		if lower == "false" || lower == "0" {
			return false
		}
		return phpTruthyString(v)
	case []byte:
		return RestSanitizeBoolean(string(v))
	default:
		return ValidateBoolean(value)
	}
}

// phpTruthyString approximates PHP (bool) string casting: only "" and "0" are false.
func phpTruthyString(s string) bool {
	return s != "" && s != "0"
}
