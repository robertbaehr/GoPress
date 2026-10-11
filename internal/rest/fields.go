// register_rest_field (P9.26) — subset of wp-includes/rest-api.php.
package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// RestFieldArgs holds callbacks/schema for a registered REST field (register_rest_field $args).
type RestFieldArgs struct {
	GetCallback    RestFieldGetCallback
	UpdateCallback RestFieldUpdateCallback
	Schema         map[string]any
}

// RestFieldGetCallback mirrors get_callback( $object, $field_name, $request, $object_type ).
// object is the prepared response map (or typed source); request may be nil in unit tests.
type RestFieldGetCallback func(object map[string]any, fieldName string, r *http.Request, objectType string) any

// RestFieldUpdateCallback mirrors update_callback( $value, $object, $field_name ).
type RestFieldUpdateCallback func(value any, object map[string]any, fieldName string) error

var (
	restFieldsMu sync.RWMutex
	restFields   = map[string]map[string]RestFieldArgs{} // object_type → attribute → args
)

// RegisterRestField registers an additional field on one or more object types
// (register_rest_field). objectType may be a single type or a slice of types.
func RegisterRestField(objectType any, attribute string, args RestFieldArgs) {
	attribute = wp.SanitizeKey(attribute)
	if attribute == "" {
		return
	}
	types := normalizeRestFieldObjectTypes(objectType)
	if len(types) == 0 {
		return
	}
	restFieldsMu.Lock()
	defer restFieldsMu.Unlock()
	for _, t := range types {
		if restFields[t] == nil {
			restFields[t] = map[string]RestFieldArgs{}
		}
		restFields[t][attribute] = args
	}
}

// ResetRestFields clears the registry (tests).
func ResetRestFields() {
	restFieldsMu.Lock()
	defer restFieldsMu.Unlock()
	restFields = map[string]map[string]RestFieldArgs{}
}

// RestFieldNames returns registered attribute names for objectType (stable test helper).
func RestFieldNames(objectType string) []string {
	objectType = wp.SanitizeKey(objectType)
	restFieldsMu.RLock()
	defer restFieldsMu.RUnlock()
	m := restFields[objectType]
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalizeRestFieldObjectTypes(objectType any) []string {
	switch t := objectType.(type) {
	case string:
		k := wp.SanitizeKey(t)
		if k == "" {
			return nil
		}
		return []string{k}
	case []string:
		var out []string
		seen := map[string]bool{}
		for _, s := range t {
			k := wp.SanitizeKey(s)
			if k == "" || seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, k)
		}
		return out
	case []any:
		var out []string
		seen := map[string]bool{}
		for _, v := range t {
			k := wp.SanitizeKey(stringify(v))
			if k == "" || seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, k)
		}
		return out
	default:
		return nil
	}
}

// HasRestFields reports whether objectType has any registered fields.
func HasRestFields(objectType string) bool {
	objectType = wp.SanitizeKey(objectType)
	restFieldsMu.RLock()
	defer restFieldsMu.RUnlock()
	return len(restFields[objectType]) > 0
}

// ApplyAdditionalFields merges registered get_callback values into data
// (WP_REST_Controller::add_additional_fields_to_object subset).
func ApplyAdditionalFields(objectType string, data map[string]any, r *http.Request) map[string]any {
	if data == nil {
		data = map[string]any{}
	}
	objectType = wp.SanitizeKey(objectType)
	restFieldsMu.RLock()
	fields := restFields[objectType]
	restFieldsMu.RUnlock()
	if len(fields) == 0 {
		return data
	}
	for name, args := range fields {
		if args.GetCallback == nil {
			continue
		}
		data[name] = args.GetCallback(data, name, r, objectType)
	}
	return data
}

// EncodeWithAdditionalFields JSON-encodes v, merging register_rest_field values
// when objectType has registrations. When none are registered, returns v unchanged
// so typed structs keep their normal JSON shape.
func EncodeWithAdditionalFields(objectType string, v any, r *http.Request) any {
	if !HasRestFields(objectType) {
		return v
	}
	switch t := v.(type) {
	case map[string]any:
		return ApplyAdditionalFields(objectType, t, r)
	case []Post:
		out := make([]map[string]any, 0, len(t))
		for i := range t {
			out = append(out, ApplyAdditionalFields(objectType, structToMap(t[i]), r))
		}
		return out
	case []*Post:
		out := make([]map[string]any, 0, len(t))
		for _, p := range t {
			if p == nil {
				continue
			}
			out = append(out, ApplyAdditionalFields(objectType, structToMap(*p), r))
		}
		return out
	case Post:
		return ApplyAdditionalFields(objectType, structToMap(t), r)
	case *Post:
		if t == nil {
			return v
		}
		return ApplyAdditionalFields(objectType, structToMap(*t), r)
	default:
		m := structToMap(v)
		if m == nil {
			return v
		}
		return ApplyAdditionalFields(objectType, m, r)
	}
}

func structToMap(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
