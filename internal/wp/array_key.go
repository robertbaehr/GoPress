package wp

import (
	"fmt"
	"reflect"
	"sort"
)

// ArrayKeyFirst returns the first key of an array/map (array_key_first / wp_array_key_first, P2.46).
//
// For slices/arrays the key is the int index 0. For maps, keys are compared as
// strings and the lexicographically least key is returned (stable stand-in for
// PHP's insertion-ordered arrays). Empty or unsupported input yields (nil, false).
func ArrayKeyFirst(input any) (any, bool) {
	return arrayKeyEdge(input, true)
}

// ArrayKeyLast returns the last key of an array/map (array_key_last / wp_array_key_last, P2.46).
//
// For slices/arrays the key is len-1. For maps, the lexicographically greatest
// string key is returned. Empty or unsupported input yields (nil, false).
func ArrayKeyLast(input any) (any, bool) {
	return arrayKeyEdge(input, false)
}

func arrayKeyEdge(input any, first bool) (any, bool) {
	if input == nil {
		return nil, false
	}
	rv := reflect.ValueOf(input)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		n := rv.Len()
		if n == 0 {
			return nil, false
		}
		if first {
			return 0, true
		}
		return n - 1, true
	case reflect.Map:
		keys := rv.MapKeys()
		if len(keys) == 0 {
			return nil, false
		}
		type keyed struct {
			s string
			v reflect.Value
		}
		list := make([]keyed, 0, len(keys))
		for _, k := range keys {
			list = append(list, keyed{s: fmt.Sprint(k.Interface()), v: k})
		}
		sort.Slice(list, func(i, j int) bool { return list[i].s < list[j].s })
		pick := list[0]
		if !first {
			pick = list[len(list)-1]
		}
		return normalizeMapKey(pick.v.Interface()), true
	default:
		return nil, false
	}
}
