package wp

import (
	"fmt"
	"reflect"
	"strings"
)

// FilterObjectList filters a list of maps/structs by key=>value args
// (wp_filter_object_list / WP_List_Util::filter, P2.43).
//
// operator is AND (default), OR, or NOT (case-insensitive). When field is
// non-empty, matching items are plucked with ListPluck(field, "").
func FilterObjectList(input any, args map[string]any, operator string, field string) any {
	if input == nil {
		if field != "" {
			return []any{}
		}
		return []any{}
	}
	rv := reflect.ValueOf(input)
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return []any{}
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []any{}
	}

	op := strings.ToUpper(strings.TrimSpace(operator))
	if op == "" {
		op = "AND"
	}
	if op != "AND" && op != "OR" && op != "NOT" {
		return []any{}
	}

	filtered := make([]any, 0, rv.Len())
	if len(args) == 0 {
		for i := 0; i < rv.Len(); i++ {
			filtered = append(filtered, rv.Index(i).Interface())
		}
	} else {
		argCount := len(args)
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i)
			matched := 0
			for key, want := range args {
				got, ok := pluckFieldOK(elem, key)
				if !ok {
					continue
				}
				if valuesEqualLoose(got, want) {
					matched++
				}
			}
			keep := false
			switch op {
			case "AND":
				keep = matched == argCount
			case "OR":
				keep = matched > 0
			case "NOT":
				keep = matched == 0
			}
			if keep {
				filtered = append(filtered, elem.Interface())
			}
		}
	}

	if field != "" {
		return ListPluck(filtered, field, "")
	}
	return filtered
}

// ListFilter is wp_list_filter — FilterObjectList without field pluck (P2.43).
func ListFilter(input any, args map[string]any, operator string) any {
	return FilterObjectList(input, args, operator, "")
}

func valuesEqualLoose(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Mirror PHP == for common scalars (int/string/bool/float).
	as := normalizeCompare(a)
	bs := normalizeCompare(b)
	return as == bs
}

func normalizeCompare(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "1"
		}
		return ""
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case string:
		return t
	case []byte:
		return string(t)
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Bool:
			if rv.Bool() {
				return "1"
			}
			return ""
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return fmt.Sprintf("%d", rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return fmt.Sprintf("%d", rv.Uint())
		case reflect.Float32, reflect.Float64:
			f := rv.Float()
			if f == float64(int64(f)) {
				return fmt.Sprintf("%d", int64(f))
			}
			return fmt.Sprintf("%v", f)
		case reflect.String:
			return rv.String()
		default:
			return fmt.Sprint(v)
		}
	}
}
