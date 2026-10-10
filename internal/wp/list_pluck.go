package wp

import (
	"fmt"
	"reflect"
)

// ListPluck extracts a field from each element of a list (wp_list_pluck, P2.37).
//
// input may be a slice/array of maps or structs. When indexKey is empty, values
// are returned in input order (slice). When indexKey is set, results are keyed
// by that field (map[any]any), matching array_column / WP_List_Util::pluck.
func ListPluck(input any, field string, indexKey string) any {
	if field == "" || input == nil {
		if indexKey == "" {
			return []any{}
		}
		return map[any]any{}
	}
	rv := reflect.ValueOf(input)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			if indexKey == "" {
				return []any{}
			}
			return map[any]any{}
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		if indexKey == "" {
			return []any{}
		}
		return map[any]any{}
	}

	if indexKey == "" {
		out := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out = append(out, pluckField(rv.Index(i), field))
		}
		return out
	}

	out := map[any]any{}
	var appendIdx int
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)
		val := pluckField(elem, field)
		idxVal, ok := pluckFieldOK(elem, indexKey)
		if ok && idxVal != nil {
			out[normalizeMapKey(idxVal)] = val
		} else {
			out[appendIdx] = val
			appendIdx++
		}
	}
	return out
}

func pluckField(elem reflect.Value, field string) any {
	v, _ := pluckFieldOK(elem, field)
	return v
}

func pluckFieldOK(elem reflect.Value, field string) (any, bool) {
	for elem.Kind() == reflect.Interface || elem.Kind() == reflect.Pointer {
		if elem.IsNil() {
			return nil, false
		}
		elem = elem.Elem()
	}
	switch elem.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(field)
		// Try string key first; also accept any key that stringifies equal.
		if !key.Type().AssignableTo(elem.Type().Key()) {
			if elem.Type().Key().Kind() == reflect.String {
				key = reflect.ValueOf(field)
			} else {
				return nil, false
			}
		}
		got := elem.MapIndex(key)
		if !got.IsValid() {
			return nil, false
		}
		return got.Interface(), true
	case reflect.Struct:
		f := elem.FieldByName(field)
		if !f.IsValid() {
			// Case-insensitive / exported-only fallback via name match.
			t := elem.Type()
			for i := 0; i < t.NumField(); i++ {
				sf := t.Field(i)
				if sf.Name == field || equalFoldASCII(sf.Name, field) {
					f = elem.Field(i)
					break
				}
			}
		}
		if !f.IsValid() || !f.CanInterface() {
			return nil, false
		}
		return f.Interface(), true
	default:
		return nil, false
	}
}

func normalizeMapKey(v any) any {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		if t == float64(int(t)) {
			return int(t)
		}
		return t
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
