package wp

import (
	"fmt"
	"reflect"
	"strconv"
)

// ArraySliceAssoc extracts a subset of associative array keys (wp_array_slice_assoc, P2.38 / P2.88).
//
// input may be a map (string, int, int64, or any keys), a struct, or a pointer to either.
// Keys are matched as strings. Struct fields also match json tags and case-insensitive
// names (via pluckFieldOK / P2.87). Missing keys are skipped. Present keys with nil
// values are included. Returns a new map[string]any (never mutates input).
func ArraySliceAssoc(input any, keys []string) map[string]any {
	out := map[string]any{}
	if input == nil || len(keys) == 0 {
		return out
	}
	rv := reflect.ValueOf(input)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return out
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		for _, key := range keys {
			if v, ok := mapLookupString(rv, key); ok {
				out[key] = v
			}
		}
	case reflect.Struct:
		for _, key := range keys {
			if v, ok := pluckFieldOK(rv, key); ok {
				out[key] = v
			}
		}
	}
	return out
}

func mapLookupString(m reflect.Value, key string) (any, bool) {
	kt := m.Type().Key()
	switch kt.Kind() {
	case reflect.String:
		got := m.MapIndex(reflect.ValueOf(key))
		if !got.IsValid() {
			return nil, false
		}
		return got.Interface(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// P2.88: numeric map keys match stringified digits ("0", "12").
		n, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return nil, false
		}
		kv := reflect.ValueOf(n).Convert(kt)
		got := m.MapIndex(kv)
		if !got.IsValid() {
			return nil, false
		}
		return got.Interface(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			return nil, false
		}
		kv := reflect.ValueOf(n).Convert(kt)
		got := m.MapIndex(kv)
		if !got.IsValid() {
			return nil, false
		}
		return got.Interface(), true
	default:
		// map[any]any and other key types: try string key, then stringify scan.
		if kt.Kind() == reflect.Interface {
			got := m.MapIndex(reflect.ValueOf(any(key)))
			if got.IsValid() {
				return got.Interface(), true
			}
			if n, err := strconv.Atoi(key); err == nil {
				got = m.MapIndex(reflect.ValueOf(any(n)))
				if got.IsValid() {
					return got.Interface(), true
				}
				got = m.MapIndex(reflect.ValueOf(any(int64(n))))
				if got.IsValid() {
					return got.Interface(), true
				}
			}
		}
		for _, mk := range m.MapKeys() {
			if fmt.Sprint(mk.Interface()) == key {
				v := m.MapIndex(mk)
				if v.IsValid() {
					return v.Interface(), true
				}
			}
		}
		return nil, false
	}
}
