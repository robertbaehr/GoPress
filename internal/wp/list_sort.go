package wp

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// ListSort sorts a list of maps/structs by fields (wp_list_sort / WP_List_Util::sort, P2.44).
//
// orderby may be a string field name or map[string]string of field→ASC|DESC.
// When orderby is a string, order is ASC (default) or DESC.
// preserveKeys is accepted for API parity; Go slices have no string keys, so it is a no-op.
func ListSort(input any, orderby any, order string, preserveKeys bool) any {
	_ = preserveKeys
	if input == nil {
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

	fields := normalizeOrderby(orderby, order)
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	if len(fields) == 0 {
		return out
	}

	sort.SliceStable(out, func(i, j int) bool {
		a := reflect.ValueOf(out[i])
		b := reflect.ValueOf(out[j])
		for _, f := range fields {
			av, aok := pluckFieldOK(a, f.field)
			bv, bok := pluckFieldOK(b, f.field)
			if !aok || !bok {
				continue
			}
			cmp := compareSortValues(av, bv)
			if cmp == 0 {
				continue
			}
			if f.desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
	return out
}

type sortField struct {
	field string
	desc  bool
}

func normalizeOrderby(orderby any, order string) []sortField {
	dir := strings.ToUpper(strings.TrimSpace(order))
	if dir != "DESC" {
		dir = "ASC"
	}
	switch v := orderby.(type) {
	case nil:
		return nil
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		return []sortField{{field: v, desc: dir == "DESC"}}
	case []string:
		out := make([]sortField, 0, len(v))
		for _, field := range v {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			out = append(out, sortField{field: field, desc: dir == "DESC"})
		}
		return out
	case map[string]string:
		// Deterministic key order (Go maps are unordered; WP arrays preserve insertion).
		keys := make([]string, 0, len(v))
		for field := range v {
			keys = append(keys, field)
		}
		sort.Strings(keys)
		out := make([]sortField, 0, len(keys))
		for _, field := range keys {
			f := strings.TrimSpace(field)
			if f == "" {
				continue
			}
			out = append(out, sortField{field: f, desc: strings.EqualFold(strings.TrimSpace(v[field]), "DESC")})
		}
		return out
	case map[string]any:
		keys := make([]string, 0, len(v))
		for field := range v {
			keys = append(keys, field)
		}
		sort.Strings(keys)
		out := make([]sortField, 0, len(keys))
		for _, field := range keys {
			f := strings.TrimSpace(field)
			if f == "" {
				continue
			}
			ds := strings.TrimSpace(fmt.Sprint(v[field]))
			out = append(out, sortField{field: f, desc: strings.EqualFold(ds, "DESC")})
		}
		return out
	default:
		return nil
	}
}

func compareSortValues(a, b any) int {
	if an, aok := toSortFloat(a); aok {
		if bn, bok := toSortFloat(b); bok {
			switch {
			case an < bn:
				return -1
			case an > bn:
				return 1
			default:
				return 0
			}
		}
	}
	as := fmt.Sprint(a)
	bs := fmt.Sprint(b)
	return strings.Compare(as, bs)
}

func toSortFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint64:
		return float64(t), true
	case float32:
		return float64(t), true
	case float64:
		return t, true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
