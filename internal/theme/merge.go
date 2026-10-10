package theme

import (
	"fmt"
)

// Merge returns a new JSON combining base with incoming (WP_Theme_JSON::merge subset, P9.20).
//
// Maps are merged recursively. Leaf arrays (and non-map scalars) from incoming
// replace base values — matching WP's overwrite of presets / spacing.units rather
// than index-wise array_replace_recursive. Version becomes max(base, incoming).
// FileSHA256 is cleared; SettingsStylesSHA256 is recomputed.
func Merge(base, incoming *JSON) (*JSON, error) {
	if base == nil && incoming == nil {
		return nil, fmt.Errorf("theme: merge nil")
	}
	if base == nil {
		return cloneJSON(incoming)
	}
	if incoming == nil {
		return cloneJSON(base)
	}
	out := &JSON{
		Theme:    firstNonEmpty(incoming.Theme, base.Theme),
		Version:  base.Version,
		Settings: mergeMaps(base.Settings, incoming.Settings),
		Styles:   mergeMaps(base.Styles, incoming.Styles),
		Path:     firstNonEmpty(incoming.Path, base.Path),
	}
	if incoming.Version > out.Version {
		out.Version = incoming.Version
	}
	sum, err := hashSettingsStyles(out.Settings, out.Styles)
	if err != nil {
		return nil, err
	}
	out.SettingsStylesSHA256 = sum
	return out, nil
}

// MergeLayers merges origin layers default → theme → custom (get_merged_data subset, P9.20).
// Nil layers are skipped. The rightmost non-nil theme slug/path wins.
func MergeLayers(layers ...*JSON) (*JSON, error) {
	var out *JSON
	for _, layer := range layers {
		if layer == nil {
			continue
		}
		merged, err := Merge(out, layer)
		if err != nil {
			return nil, err
		}
		out = merged
	}
	if out == nil {
		return nil, fmt.Errorf("theme: merge layers empty")
	}
	return out, nil
}

func cloneJSON(in *JSON) (*JSON, error) {
	if in == nil {
		return nil, fmt.Errorf("theme: clone nil")
	}
	out := &JSON{
		Theme:    in.Theme,
		Version:  in.Version,
		Settings: cloneMap(in.Settings),
		Styles:   cloneMap(in.Styles),
		Path:     in.Path,
	}
	sum, err := hashSettingsStyles(out.Settings, out.Styles)
	if err != nil {
		return nil, err
	}
	out.SettingsStylesSHA256 = sum
	return out, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// mergeMaps deep-merges src into a clone of dst. Leaf arrays are replaced.
func mergeMaps(dst, src map[string]any) map[string]any {
	out := cloneMap(dst)
	if src == nil {
		return out
	}
	if out == nil {
		out = map[string]any{}
	}
	for k, sv := range src {
		dv, ok := out[k]
		srcMap, srcIsMap := asMap(sv)
		dstMap, dstIsMap := asMap(dv)
		if srcIsMap && ok && dstIsMap {
			out[k] = mergeMaps(dstMap, srcMap)
			continue
		}
		out[k] = cloneValue(sv)
	}
	return out
}

func asMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return cloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i, el := range t {
			out[i] = cloneValue(el)
		}
		return out
	default:
		return v
	}
}
