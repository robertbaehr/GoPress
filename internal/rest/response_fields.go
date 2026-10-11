package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// applyRESTResponseFields filters v by ?_fields= when set (rest_filter_response_fields, P4.147).
// Errors and empty field lists are returned unchanged. Works on maps, slices of maps,
// and JSON-marshalable structs (via structToMap).
func applyRESTResponseFields(r *http.Request, v any) any {
	if r == nil || r.URL == nil || v == nil {
		return v
	}
	raw := strings.TrimSpace(r.URL.Query().Get("_fields"))
	if raw == "" {
		return v
	}
	fields := wp.ParseList(raw)
	if len(fields) == 0 {
		return v
	}
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	keyed := fieldsAsKeyedTree(fields)
	return filterValueByFields(v, keyed)
}

// fieldsAsKeyedTree builds the nested inclusion tree from dotted field names
// (see rest_filter_response_fields in wp-includes/rest-api.php).
func fieldsAsKeyedTree(fields []string) map[string]any {
	out := map[string]any{}
	for _, field := range fields {
		if field == "" {
			continue
		}
		parts := strings.Split(field, ".")
		ref := out
		skip := false
		for len(parts) > 1 {
			next := parts[0]
			parts = parts[1:]
			if existing, ok := ref[next]; ok {
				if existing == true {
					skip = true
					break
				}
			} else {
				ref[next] = map[string]any{}
			}
			child, ok := ref[next].(map[string]any)
			if !ok {
				skip = true
				break
			}
			ref = child
		}
		if skip {
			continue
		}
		last := parts[0]
		ref[last] = true
	}
	return out
}

func filterValueByFields(v any, keyed map[string]any) any {
	switch t := v.(type) {
	case map[string]any:
		return restArrayIntersectKeyRecursive(t, keyed)
	case []map[string]any:
		out := make([]map[string]any, len(t))
		for i := range t {
			out[i] = restArrayIntersectKeyRecursive(t[i], keyed)
		}
		return out
	case []any:
		if !wp.IsNumericArray(t) {
			m := structToMap(v)
			if m == nil {
				return v
			}
			return restArrayIntersectKeyRecursive(m, keyed)
		}
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, filterValueByFields(item, keyed))
		}
		return out
	case []Post:
		out := make([]map[string]any, 0, len(t))
		for i := range t {
			m := structToMap(t[i])
			out = append(out, restArrayIntersectKeyRecursive(m, keyed))
		}
		return out
	case []*Post:
		out := make([]map[string]any, 0, len(t))
		for _, p := range t {
			if p == nil {
				continue
			}
			m := structToMap(*p)
			out = append(out, restArrayIntersectKeyRecursive(m, keyed))
		}
		return out
	case Post:
		return restArrayIntersectKeyRecursive(structToMap(t), keyed)
	case *Post:
		if t == nil {
			return v
		}
		return restArrayIntersectKeyRecursive(structToMap(*t), keyed)
	default:
		m := structToMap(v)
		if m == nil {
			return v
		}
		// Numeric JSON arrays unmarshal as []any — handle via filterValueByFields on []any.
		if wp.IsNumericArray(m) {
			return v
		}
		return restArrayIntersectKeyRecursive(m, keyed)
	}
}

// restArrayIntersectKeyRecursive mirrors _rest_array_intersect_key_recursive.
func restArrayIntersectKeyRecursive(data map[string]any, keyed map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	out := map[string]any{}
	for k, want := range keyed {
		val, ok := data[k]
		if !ok {
			continue
		}
		if wantMap, ok := want.(map[string]any); ok {
			if child, ok := val.(map[string]any); ok {
				out[k] = restArrayIntersectKeyRecursive(child, wantMap)
				continue
			}
		}
		out[k] = val
	}
	return out
}
