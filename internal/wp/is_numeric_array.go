package wp

import "reflect"

// IsNumericArray reports whether data is a list-like array whose keys are all
// non-string (wp_is_numeric_array, P2.36).
//
// In Go: slices/arrays are numeric; maps are numeric only when every key is an
// integer kind (empty maps count as numeric). Non-array values return false.
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
			switch k.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				continue
			default:
				// PHP rejects string keys (even digit strings typed as string).
				return false
			}
		}
		return true
	default:
		return false
	}
}
