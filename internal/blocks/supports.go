package blocks

import (
	"fmt"
	"html"
	"strings"
)

// applySupports injects basic color/spacing/fontSize/layout/shadow/position classes and styles
// from block attributes onto the rendered root element (P5.10 / P5.19 / P5.27 / P5.28 / P5.29).
func applySupports(htmlFragment string, attrs map[string]any, blockName string) string {
	if htmlFragment == "" {
		return htmlFragment
	}
	classes, styles := supportsFromAttrs(attrs, blockName)
	out := htmlFragment
	for _, c := range classes {
		out = addRootClass(out, c)
	}
	if styles != "" {
		out = addRootStyle(out, styles)
	}
	return out
}

func supportsFromAttrs(attrs map[string]any, blockName string) (classes []string, styles string) {
	if attrs != nil {
		if cn, ok := attrs["className"].(string); ok && cn != "" {
			classes = append(classes, strings.Fields(cn)...)
		}
		if align, ok := attrs["align"].(string); ok && align != "" {
			classes = appendUnique(classes, "align"+slugifySupport(align))
		}
		if bg, ok := attrs["backgroundColor"].(string); ok && bg != "" {
			classes = append(classes, "has-background", "has-"+slugifySupport(bg)+"-background-color")
		}
		if tc, ok := attrs["textColor"].(string); ok && tc != "" {
			classes = append(classes, "has-text-color", "has-"+slugifySupport(tc)+"-color")
		}
		if fs, ok := attrs["fontSize"].(string); ok && fs != "" {
			classes = append(classes, "has-"+slugifySupport(fs)+"-font-size")
		}
		if ta, ok := attrs["textAlign"].(string); ok && ta != "" {
			classes = appendUnique(classes, "has-text-align-"+slugifySupport(ta))
		}

		if style, ok := attrs["style"].(map[string]any); ok {
			if color, ok := style["color"].(map[string]any); ok {
				if bg, ok := color["background"].(string); ok && bg != "" {
					classes = appendUnique(classes, "has-background")
				}
				if tx, ok := color["text"].(string); ok && tx != "" {
					classes = appendUnique(classes, "has-text-color")
				}
				if grad, ok := color["gradient"].(string); ok && grad != "" {
					classes = appendUnique(classes, "has-background")
				}
			}
			styles = CompileStyle(style)
			if filterID, filterCSS := duotoneFromStyle(style); filterID != "" {
				classes = appendUnique(classes, filterID)
				if filterCSS != "" {
					styles = mergeCSSDecls(styles, filterCSS)
				}
			}
			if pos, ok := style["position"].(map[string]any); ok {
				if t, ok := pos["type"].(string); ok {
					t = strings.TrimSpace(t)
					if t == "sticky" || t == "fixed" {
						classes = appendUnique(classes, "is-position-"+t)
					}
				}
			}
		}
		// Top-level shadow attribute (registered by wp_register_shadow_support).
		if shadow, ok := attrs["shadow"].(string); ok && shadow != "" {
			styles = mergeCSSDecls(styles, "box-shadow:"+cssValue(shadow))
		}
	}
	classes = append(classes, layoutClasses(blockName, attrs)...)
	if layoutCSS := layoutInlineStyles(blockName, attrs); layoutCSS != "" {
		styles = mergeCSSDecls(styles, layoutCSS)
	}
	return classes, styles
}

// duotoneFromStyle extracts WP duotone support (P5.22): class wp-duotone-{slug}
// plus an inline filter declaration for preset / unset values.
// Attr paths: style.color.duotone (legacy) and style.filter.duotone (current).
func duotoneFromStyle(style map[string]any) (filterID, filterCSS string) {
	attr := duotoneAttr(style)
	if attr == nil {
		return "", ""
	}
	switch v := attr.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return "", ""
		}
		if slug := duotoneSlugFromAttr(v); slug != "" {
			filterID = "wp-duotone-" + slug
			filterCSS = "filter:var(--wp--preset--duotone--" + slug + ")"
			return filterID, filterCSS
		}
		// CSS values like "unset".
		slug := slugifySupport(v)
		if slug == "" {
			slug = "custom"
		}
		return "wp-duotone-" + slug, "filter:" + html.EscapeString(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, c := range v {
			s, ok := c.(string)
			if !ok {
				continue
			}
			s = strings.TrimPrefix(strings.TrimSpace(s), "#")
			if s != "" {
				parts = append(parts, slugifySupport(s))
			}
		}
		if len(parts) == 0 {
			return "", ""
		}
		return "wp-duotone-" + strings.Join(parts, "-"), ""
	default:
		return "", ""
	}
}

func duotoneAttr(style map[string]any) any {
	if style == nil {
		return nil
	}
	if filter, ok := style["filter"].(map[string]any); ok {
		if d, ok := filter["duotone"]; ok && d != nil {
			return d
		}
	}
	if color, ok := style["color"].(map[string]any); ok {
		if d, ok := color["duotone"]; ok && d != nil {
			return d
		}
	}
	return nil
}

func duotoneSlugFromAttr(attr string) string {
	attr = strings.TrimSpace(attr)
	if strings.HasPrefix(attr, "var:preset|duotone|") {
		return slugifySupport(strings.TrimPrefix(attr, "var:preset|duotone|"))
	}
	const prefix = "var(--wp--preset--duotone--"
	if strings.HasPrefix(attr, prefix) && strings.HasSuffix(attr, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(attr, prefix), ")")
		return slugifySupport(inner)
	}
	return ""
}

// layoutClasses mirrors WP block layout support classnames (P5.19 / P5.29).
func layoutClasses(blockName string, attrs map[string]any) []string {
	info := layoutInfoFromAttrs(blockName, attrs)
	if info.Type == "" {
		return nil
	}
	slug := info.Type
	switch info.Type {
	case "default":
		slug = "flow"
	}
	className := "is-layout-" + slug
	var out []string
	if info.FromAttrs || blockHasDefaultLayout(blockName) {
		if slug == "constrained" {
			out = append(out, "has-global-padding")
		}
		if info.Orientation != "" {
			out = append(out, "is-"+slugifySupport(info.Orientation))
		}
		if info.Justify != "" {
			out = append(out, "is-content-justification-"+slugifySupport(info.Justify))
		}
		if info.FlexWrap == "nowrap" {
			out = append(out, "is-nowrap")
		}
		// verticalAlignment maps to CSS align-items/justify-content only (no classname in WP).
		out = append(out, className)
		if bn := blockClassStem(blockName); bn != "" {
			out = append(out, bn+"-is-layout-"+slug)
		}
	}
	return out
}

// layoutInlineStyles emits root-level flex/grid CSS when layout attrs are set (P5.29).
// WP stores these via the style engine; GoPress applies them inline on the wrapper.
// Only runs for explicit layout attrs so default block layouts stay class-only (front HTML parity).
func layoutInlineStyles(blockName string, attrs map[string]any) string {
	if attrs == nil {
		return ""
	}
	layout, ok := attrs["layout"].(map[string]any)
	if !ok || layout == nil {
		return ""
	}
	info := layoutInfoFromAttrs(blockName, attrs)
	var parts []string
	switch info.Type {
	case "flex":
		parts = append(parts, "display:flex")
		orient := info.Orientation
		if orient == "" {
			orient = "horizontal"
		}
		justifyMap := map[string]string{
			"left": "flex-start", "right": "flex-end", "center": "center",
			"space-between": "space-between", "stretch": "stretch",
		}
		alignMap := map[string]string{
			"top": "flex-start", "center": "center", "bottom": "flex-end",
			"stretch": "stretch", "space-between": "space-between",
		}
		if orient == "vertical" {
			parts = append(parts, "flex-direction:column")
			if j, ok := justifyMap[info.Justify]; ok {
				parts = append(parts, "align-items:"+j)
			} else if info.Justify == "" {
				parts = append(parts, "align-items:flex-start")
			}
			if a, ok := alignMap[info.VerticalAlign]; ok {
				parts = append(parts, "justify-content:"+a)
			}
		} else {
			if j, ok := justifyMap[info.Justify]; ok {
				parts = append(parts, "justify-content:"+j)
			}
			if a, ok := alignMap[info.VerticalAlign]; ok {
				parts = append(parts, "align-items:"+a)
			}
		}
		if info.FlexWrap == "nowrap" {
			parts = append(parts, "flex-wrap:nowrap")
		}
	case "grid":
		parts = append(parts, "display:grid")
		if info.ColumnCount > 0 && info.MinimumColumnWidth != "" {
			parts = append(parts, fmt.Sprintf(
				"grid-template-columns:repeat(auto-fill, minmax(min(%s, 100%%), 1fr))",
				cssValue(info.MinimumColumnWidth),
			))
		} else if info.ColumnCount > 0 {
			parts = append(parts, fmt.Sprintf(
				"grid-template-columns:repeat(%d, minmax(0, 1fr))", info.ColumnCount,
			))
		} else if info.MinimumColumnWidth != "" {
			parts = append(parts, fmt.Sprintf(
				"grid-template-columns:repeat(auto-fill, minmax(min(%s, 100%%), 1fr))",
				cssValue(info.MinimumColumnWidth),
			))
		}
	default:
		return ""
	}
	return strings.Join(parts, ";")
}

type layoutInfo struct {
	Type               string
	Justify            string
	Orientation        string
	FlexWrap           string
	VerticalAlign      string
	ColumnCount        int
	MinimumColumnWidth string
	FromAttrs          bool
}

func layoutInfoFromAttrs(blockName string, attrs map[string]any) layoutInfo {
	var info layoutInfo
	if attrs != nil {
		if layout, ok := attrs["layout"].(map[string]any); ok && layout != nil {
			info.FromAttrs = true
			if t, ok := layout["type"].(string); ok {
				info.Type = strings.TrimSpace(t)
			}
			if j, ok := layout["justifyContent"].(string); ok {
				info.Justify = strings.TrimSpace(j)
			}
			if o, ok := layout["orientation"].(string); ok {
				info.Orientation = strings.TrimSpace(o)
			}
			if w, ok := layout["flexWrap"].(string); ok {
				info.FlexWrap = strings.TrimSpace(w)
			}
			if v, ok := layout["verticalAlignment"].(string); ok {
				info.VerticalAlign = strings.TrimSpace(v)
			}
			switch n := layout["columnCount"].(type) {
			case float64:
				if n > 0 {
					info.ColumnCount = int(n)
				}
			case int:
				if n > 0 {
					info.ColumnCount = n
				}
			}
			if w, ok := layout["minimumColumnWidth"].(string); ok {
				info.MinimumColumnWidth = strings.TrimSpace(w)
			}
		}
	}
	if info.Type == "" {
		info.Type = defaultLayoutType(blockName)
	}
	if info.Type == "" {
		return layoutInfo{}
	}
	if !info.FromAttrs {
		info.FromAttrs = blockHasDefaultLayout(blockName)
	}
	return info
}

func defaultLayoutType(blockName string) string {
	switch blockName {
	case "core/columns", "core/navigation", "core/buttons":
		return "flex"
	case "core/column", "core/group", "core/query", "core/post-template", "core/post-content", "core/quote":
		return "default" // → flow
	default:
		return ""
	}
}

func blockHasDefaultLayout(blockName string) bool {
	return defaultLayoutType(blockName) != ""
}

func blockClassStem(blockName string) string {
	blockName = strings.TrimSpace(blockName)
	if blockName == "" {
		return ""
	}
	name := blockName
	if i := strings.Index(blockName, "/"); i >= 0 {
		name = blockName[i+1:]
	}
	return "wp-block-" + name
}

func spacingStyles(prop string, v any) []string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{prop + ":" + cssValue(t)}
	case map[string]any:
		var out []string
		for _, side := range []string{"top", "right", "bottom", "left"} {
			if s, ok := t[side].(string); ok && s != "" {
				out = append(out, prop+"-"+side+":"+cssValue(s))
			}
		}
		return out
	default:
		return nil
	}
}

func cssValue(v string) string {
	v = strings.TrimSpace(v)
	// WP preset vars: var:preset|spacing|50 → var(--wp--preset--spacing--50)
	if strings.HasPrefix(v, "var:") {
		inner := strings.TrimPrefix(v, "var:")
		parts := strings.Split(inner, "|")
		return "var(--wp--" + strings.Join(parts, "--") + ")"
	}
	return html.EscapeString(v)
}

func slugifySupport(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '_', r == '/':
			b.WriteByte('-')
		case r == '-':
			b.WriteByte('-')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// mergeCSSDecls appends declarations from add whose property keys are not already in existing.
func mergeCSSDecls(existing, add string) string {
	existing = strings.TrimSpace(existing)
	add = strings.TrimSpace(add)
	if add == "" {
		return existing
	}
	have := map[string]bool{}
	for _, part := range strings.Split(existing, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.Index(part, ":"); i > 0 {
			have[strings.ToLower(strings.TrimSpace(part[:i]))] = true
		}
	}
	var extra []string
	for _, part := range strings.Split(add, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		prop := part
		if i := strings.Index(part, ":"); i > 0 {
			prop = strings.ToLower(strings.TrimSpace(part[:i]))
		}
		if have[prop] {
			continue
		}
		extra = append(extra, part)
		have[prop] = true
	}
	if len(extra) == 0 {
		return existing
	}
	out := existing
	if out != "" && !strings.HasSuffix(out, ";") {
		out += ";"
	}
	return out + strings.Join(extra, ";")
}

// addRootStyle merges a CSS style string onto the first HTML opening tag.
func addRootStyle(htmlFragment, style string) string {
	if htmlFragment == "" || style == "" {
		return htmlFragment
	}
	i := strings.Index(htmlFragment, "<")
	if i < 0 {
		return htmlFragment
	}
	rest := htmlFragment[i+1:]
	if rest == "" || rest[0] == '/' || rest[0] == '!' {
		return htmlFragment
	}
	end := strings.Index(rest, ">")
	if end < 0 {
		return htmlFragment
	}
	open := rest[:end]
	selfClose := strings.HasSuffix(open, "/")
	if selfClose {
		open = strings.TrimSpace(strings.TrimSuffix(open, "/"))
	}
	lower := strings.ToLower(open)
	if idx := strings.Index(lower, `style="`); idx >= 0 {
		valStart := idx + len(`style="`)
		valEnd := strings.Index(open[valStart:], `"`)
		if valEnd < 0 {
			return htmlFragment
		}
		valEnd += valStart
		existing := open[valStart:valEnd]
		merged := mergeCSSDecls(existing, style)
		open = open[:valStart] + merged + open[valEnd:]
	} else {
		open = open + fmt.Sprintf(` style="%s"`, style)
	}
	suffix := ">"
	if selfClose {
		suffix = " />"
	}
	return htmlFragment[:i+1] + open + suffix + rest[end+1:]
}
