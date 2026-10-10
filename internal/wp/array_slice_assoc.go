package wp

import (
	"fmt"
	"reflect"
)

// ArraySliceAssoc extracts a subset of associative array keys (wp_array_slice_assoc, P2.38).
//
// input may be a map (string or any keys) or a struct; keys are matched as strings.
// Missing keys are skipped. Returns a new map[string]any (never mutates input).
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
	default:
		got := m.MapIndex(reflect.ValueOf(any(key)))
		if got.IsValid() {
			return got.Interface(), true
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
