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
		parts = append(parts, typographyStyles(typo)...)
	}
	if border, ok := style["border"].(map[string]any); ok {
		parts = append(parts, borderStyles(border)...)
	}
	// Shadow block support (P5.27): style.shadow → box-shadow.
	if shadow, ok := style["shadow"].(string); ok && shadow != "" {
		parts = append(parts, "box-shadow:"+cssValue(shadow))
	}
	// Position block support (P5.28): style.position → position/z-index/sides.
	parts = append(parts, positionStyles(style["position"])...)
	return strings.Join(parts, ";")
}

// typographyStyles mirrors WP typography block support CSS (P5.30).
func typographyStyles(typo map[string]any) []string {
	if typo == nil {
		return nil
	}
	var parts []string
	add := func(cssProp, key string) {
		raw, ok := typo[key]
		if !ok || raw == nil {
			return
		}
		var s string
		switch t := raw.(type) {
		case string:
			s = strings.TrimSpace(t)
		case float64:
			s = fmt.Sprintf("%g", t)
		default:
			s = strings.TrimSpace(fmt.Sprint(raw))
		}
		if s == "" {
			return
		}
		parts = append(parts, cssProp+":"+cssValue(s))
	}
	add("font-size", "fontSize")
	add("font-family", "fontFamily")
	add("font-style", "fontStyle")
	add("font-weight", "fontWeight")
	add("letter-spacing", "letterSpacing")
	add("line-height", "lineHeight")
	add("text-decoration", "textDecoration")
	add("text-transform", "textTransform")
	add("writing-mode", "writingMode")
	add("text-indent", "textIndent")
	add("column-count", "textColumns")
	add("text-shadow", "textShadow")
	return parts
}

// positionStyles mirrors wp_get_position_support_styles for sticky/fixed (P5.28).
func positionStyles(v any) []string {
	pos, ok := v.(map[string]any)
	if !ok || pos == nil {
		return nil
	}
	posType, _ := pos["type"].(string)
	posType = strings.TrimSpace(posType)
	if posType != "sticky" && posType != "fixed" {
		return nil
	}
	var out []string
	for _, side := range []string{"top", "right", "bottom", "left"} {
		raw, ok := pos[side]
		if !ok || raw == nil {
			continue
		}
		sideVal := ""
		switch t := raw.(type) {
		case string:
			sideVal = strings.TrimSpace(t)
		case float64:
			if t == 0 {
				sideVal = "0px"
			} else {
				sideVal = fmt.Sprintf("%g", t)
			}
		default:
			sideVal = strings.TrimSpace(fmt.Sprint(raw))
		}
		if sideVal == "" {
			continue
		}
		if sideVal == "0" {
			sideVal = "0px"
		}
		if side == "top" {
			sideVal = "calc(" + cssValue(sideVal) + " + var(--wp-admin--admin-bar--position-offset, 0px))"
			out = append(out, "top:"+sideVal)
			continue
		}
		out = append(out, side+":"+cssValue(sideVal))
	}
	out = append(out, "position:"+posType, "z-index:10")
	return out
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

// borderStyles mirrors WP border block support / style engine (P5.31).
func borderStyles(border map[string]any) []string {
	if border == nil {
		return nil
	}
	var out []string
	if v := borderScalar(border["color"]); v != "" {
		out = append(out, "border-color:"+cssValue(v))
	}
	if v := borderScalar(border["width"]); v != "" {
		out = append(out, "border-width:"+cssValue(maybePx(v)))
	}
	if v := borderScalar(border["style"]); v != "" {
		out = append(out, "border-style:"+cssValue(v))
	}
	switch radius := border["radius"].(type) {
	case string:
		if radius != "" {
			out = append(out, "border-radius:"+cssValue(maybePx(radius)))
		}
	case float64:
		out = append(out, "border-radius:"+cssValue(maybePx(fmt.Sprintf("%g", radius))))
	case map[string]any:
		for _, corner := range []string{"topLeft", "topRight", "bottomRight", "bottomLeft"} {
			if v := borderScalar(radius[corner]); v != "" {
				prop := "border-" + camelToKebab(corner) + "-radius"
				out = append(out, prop+":"+cssValue(maybePx(v)))
			}
		}
	}
	// Individual sides: style.border.{top|right|bottom|left}.{color|width|style}.
	for _, side := range []string{"top", "right", "bottom", "left"} {
		sideMap, ok := border[side].(map[string]any)
		if !ok || sideMap == nil {
			continue
		}
		if v := borderScalar(sideMap["color"]); v != "" {
			out = append(out, "border-"+side+"-color:"+cssValue(v))
		}
		if v := borderScalar(sideMap["width"]); v != "" {
			out = append(out, "border-"+side+"-width:"+cssValue(maybePx(v)))
		}
		if v := borderScalar(sideMap["style"]); v != "" {
			out = append(out, "border-"+side+"-style:"+cssValue(v))
		}
	}
	return out
}

func borderScalar(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return fmt.Sprintf("%g", t)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

// maybePx appends px to bare numeric border widths/radii (WP unitless legacy).
func maybePx(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	for _, r := range v {
		if (r < '0' || r > '9') && r != '.' && r != '-' {
			return v
		}
	}
	return v + "px"
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
