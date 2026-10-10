package blocks

import (
	"fmt"
	"strings"
)

// CompileStyle turns a block "style" attribute object into an inline CSS
// declarations string (WP style engine subset, P5.12).
func CompileStyle(style map[string]any) string {
	if style == nil {
		return ""
	}
	var parts []string
	if color, ok := style["color"].(map[string]any); ok {
		if bg, ok := color["background"].(string); ok && bg != "" {
			parts = append(parts, "background-color:"+cssValue(bg))
		}
		if tx, ok := color["text"].(string); ok && tx != "" {
			parts = append(parts, "color:"+cssValue(tx))
		}
		if grad, ok := color["gradient"].(string); ok && grad != "" {
			parts = append(parts, "background:"+cssValue(grad))
		}
	}
	if spacing, ok := style["spacing"].(map[string]any); ok {
		parts = append(parts, spacingStyles("padding", spacing["padding"])...)
		parts = append(parts, spacingStyles("margin", spacing["margin"])...)
		if gap, ok := spacing["blockGap"].(string); ok && gap != "" {
			parts = append(parts, "gap:"+cssValue(gap))
		}
	}
	if typo, ok := style["typography"].(map[string]any); ok {
		if fs, ok := typo["fontSize"].(string); ok && fs != "" {
			parts = append(parts, "font-size:"+cssValue(fs))
		}
		if lh, ok := typo["lineHeight"].(string); ok && lh != "" {
			parts = append(parts, "line-height:"+cssValue(lh))
		}
		if ff, ok := typo["fontFamily"].(string); ok && ff != "" {
			parts = append(parts, "font-family:"+cssValue(ff))
		}
		if fw, ok := typo["fontWeight"].(string); ok && fw != "" {
			parts = append(parts, "font-weight:"+cssValue(fw))
		}
		if ts, ok := typo["textTransform"].(string); ok && ts != "" {
			parts = append(parts, "text-transform:"+cssValue(ts))
		}
	}
	if border, ok := style["border"].(map[string]any); ok {
		parts = append(parts, borderStyles(border)...)
	}
	return strings.Join(parts, ";")
}

// CompileCSS builds a stylesheet rule for selector from a style object.
// Returns "" when there are no declarations.
func CompileCSS(selector string, style map[string]any) string {
	selector = strings.TrimSpace(selector)
	decls := CompileStyle(style)
	if selector == "" || decls == "" {
		return ""
	}
	return fmt.Sprintf("%s{%s}", selector, decls)
}

// CompileCSSRules joins multiple CompileCSS results with newlines.
func CompileCSSRules(rules []struct {
	Selector string
	Style    map[string]any
}) string {
	var b strings.Builder
	for _, r := range rules {
		rule := CompileCSS(r.Selector, r.Style)
		if rule == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(rule)
	}
	return b.String()
}

func borderStyles(border map[string]any) []string {
	var out []string
	if color, ok := border["color"].(string); ok && color != "" {
		out = append(out, "border-color:"+cssValue(color))
	}
	if width, ok := border["width"].(string); ok && width != "" {
		out = append(out, "border-width:"+cssValue(width))
	}
	if style, ok := border["style"].(string); ok && style != "" {
		out = append(out, "border-style:"+cssValue(style))
	}
	if radius, ok := border["radius"].(string); ok && radius != "" {
		out = append(out, "border-radius:"+cssValue(radius))
	} else if radiusMap, ok := border["radius"].(map[string]any); ok {
		for _, corner := range []string{"topLeft", "topRight", "bottomRight", "bottomLeft"} {
			if v, ok := radiusMap[corner].(string); ok && v != "" {
				prop := "border-" + camelToKebab(corner) + "-radius"
				out = append(out, prop+":"+cssValue(v))
			}
		}
	}
	return out
}

func camelToKebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r - 'A' + 'a')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
