package wp

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// RecursiveKSort sorts a map by key, recursively sorting nested maps
// (wp_recursive_ksort, P2.45 / P2.92). Slice elements that are maps are sorted in
// place within a new slice; non-map values are left untouched.
// Returns a new value; the input is not mutated.
//
// P2.92: map[string]string / map[any]any inputs; digit-string and integer keys
// sort numerically when every key is numeric (PHP ksort SORT_REGULAR subset).
func RecursiveKSort(input any) any {
	if input == nil {
		return nil
	}
	// Fast path for common typed maps (P2.92).
	switch m := input.(type) {
	case map[string]any:
		return recursiveKSortStringAny(m)
	case map[string]string:
		out := make(map[string]any, len(m))
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sortMapKeys(keys)
		for _, k := range keys {
			out[k] = m[k]
		}
		return out
	case map[any]any:
		return recursiveKSortAnyAny(m)
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

// RecursiveKSortInPlace sorts a map[string]any in place (PHP by-reference
// wp_recursive_ksort, P2.92). Nested map[string]any values are also sorted
// in place; other nested maps are replaced with RecursiveKSort results.
func RecursiveKSortInPlace(m *map[string]any) {
	if m == nil || *m == nil {
		return
	}
	sorted := recursiveKSortStringAny(*m)
	// Rebuild the caller's map so iteration order of keys is sorted when
	// re-encoded; Go maps have no guaranteed order, but JSON/tools use key sort.
	for k := range *m {
		delete(*m, k)
	}
	keys := make([]string, 0, len(sorted))
	for k := range sorted {
		keys = append(keys, k)
	}
	sortMapKeys(keys)
	for _, k := range keys {
		v := sorted[k]
		if nested, ok := v.(map[string]any); ok {
			cp := nested
			RecursiveKSortInPlace(&cp)
			(*m)[k] = cp
			continue
		}
		(*m)[k] = v
	}
}

func recursiveKSortStringAny(m map[string]any) map[string]any {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortMapKeys(keys)
	out := make(map[string]any, len(keys))
	for _, k := range keys {
		out[k] = RecursiveKSort(m[k])
	}
	return out
}

func recursiveKSortAnyAny(m map[any]any) map[string]any {
	keys := make([]string, 0, len(m))
	keyByStr := make(map[string]any, len(m))
	for k, v := range m {
		s := stringifyAnyKey(k)
		keys = append(keys, s)
		keyByStr[s] = v
	}
	sortMapKeys(keys)
	out := make(map[string]any, len(keys))
	for _, s := range keys {
		out[s] = RecursiveKSort(keyByStr[s])
	}
	return out
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
	sortMapKeys(strKeys)
	out := make(map[string]any, len(strKeys))
	for _, s := range strKeys {
		v := rv.MapIndex(keyByStr[s]).Interface()
		out[s] = RecursiveKSort(v)
	}
	return out
}

// sortMapKeys sorts keys like PHP ksort SORT_REGULAR when every key is numeric.
func sortMapKeys(keys []string) {
	if len(keys) == 0 {
		return
	}
	allNumeric := true
	for _, k := range keys {
		if _, err := strconv.ParseInt(k, 10, 64); err != nil {
			allNumeric = false
			break
		}
	}
	if allNumeric {
		sort.SliceStable(keys, func(i, j int) bool {
			ai, _ := strconv.ParseInt(keys[i], 10, 64)
			aj, _ := strconv.ParseInt(keys[j], 10, 64)
			return ai < aj
		})
		return
	}
	sort.Strings(keys)
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

func stringifyAnyKey(k any) string {
	switch t := k.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}
