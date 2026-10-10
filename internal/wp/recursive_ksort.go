package wp

import (
	"fmt"
	"reflect"
	"sort"
)

// RecursiveKSort sorts a map by key, recursively sorting nested maps
// (wp_recursive_ksort, P2.45). Slice elements that are maps are sorted in
// place within a new slice; non-map values are left untouched.
// Returns a new value; the input is not mutated.
func RecursiveKSort(input any) any {
	if input == nil {
		return nil
	}
	rv := reflect.ValueOf(input)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		return recursiveKSortMap(rv)
	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = RecursiveKSort(rv.Index(i).Interface())
		}
		return out
	default:
		return input
	}
}

func recursiveKSortMap(rv reflect.Value) map[string]any {
	keys := rv.MapKeys()
	strKeys := make([]string, 0, len(keys))
	keyByStr := make(map[string]reflect.Value, len(keys))
	for _, k := range keys {
		s := stringifyMapKey(k)
		strKeys = append(strKeys, s)
		keyByStr[s] = k
	}
	sort.Strings(strKeys)
	out := make(map[string]any, len(strKeys))
	for _, s := range strKeys {
		v := rv.MapIndex(keyByStr[s]).Interface()
		out[s] = RecursiveKSort(v)
	}
	return out
}

func stringifyMapKey(k reflect.Value) string {
	for k.Kind() == reflect.Interface {
		if k.IsNil() {
			return ""
		}
		k = k.Elem()
	}
	if k.Kind() == reflect.String {
		return k.String()
	}
	return fmt.Sprint(k.Interface())
}
