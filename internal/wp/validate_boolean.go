// wp_validate_boolean (P2.34) — subset of wp-includes/functions.php.
package wp

import (
	"reflect"
	"strconv"
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

// phpTruthyString approximates PHP (bool) string casting.
func phpTruthyString(s string) bool {
	if s == "" || s == "0" {
		return false
	}
	// Numeric zero strings like "0.0" are truthy in PHP for strings... actually
	// only "" and "0" are false. Keep that.
	_ = strconv.IntSize
	return true
}
