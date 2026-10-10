package blocks

import (
	"fmt"
	"html"
	"strings"
	"sync/atomic"
)

// elementsSeq is the per-process counter for wp-elements-* class names (P5.32).
var elementsSeq uint64

// applySupports injects basic color/spacing/typography/layout/shadow/position/border/elements/background/dimensions/filter/aria-label
// classes, styles, and attributes from block attributes onto the rendered root element (P5.10 / P5.19 / P5.27–P5.36).
func applySupports(htmlFragment string, attrs map[string]any, blockName string) string {
	if htmlFragment == "" {
		return htmlFragment
	}
	classes, styles, elementsCSS, rootAttrs := supportsFromAttrs(attrs, blockName)
	out := htmlFragment
	for _, c := range classes {
		out = addRootClass(out, c)
	}
	if styles != "" {
		out = addRootStyle(out, styles)
	}
	for name, val := range rootAttrs {
		out = addRootAttr(out, name, val)
	}
	if elementsCSS != "" {
		out = `<style id="core-block-supports-inline-css">` + elementsCSS + `</style>` + out
	}
	return out
}

func supportsFromAttrs(attrs map[string]any, blockName string) (classes []string, styles, elementsCSS string, rootAttrs map[string]string) {
	rootAttrs = map[string]string{}
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
		if ff, ok := attrs["fontFamily"].(string); ok && ff != "" {
			classes = appendUnique(classes, "has-"+slugifySupport(ff)+"-font-family")
		}
		if ta, ok := attrs["textAlign"].(string); ok && ta != "" {
			classes = appendUnique(classes, "has-text-align-"+slugifySupport(ta))
		}
		// Top-level textShadow preset (P5.30).
		if ts, ok := attrs["textShadow"].(string); ok && ts != "" {
			styles = mergeCSSDecls(styles, "text-shadow:"+cssValue("var:preset|text-shadow|"+ts))
		}

		// Top-level borderColor preset (P5.31 / wp_register_border_support).
		if bc, ok := attrs["borderColor"].(string); ok && strings.TrimSpace(bc) != "" {
			slug := slugifySupport(bc)
			classes = appendUnique(classes, "has-border-color")
			if slug != "" {
				classes = appendUnique(classes, "has-"+slug+"-border-color")
			}
			styles = mergeCSSDecls(styles, "border-color:"+cssValue("var:preset|color|"+strings.TrimSpace(bc)))
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
			if typo, ok := style["typography"].(map[string]any); ok {
				if ta, ok := typo["textAlign"].(string); ok && ta != "" {
					classes = appendUnique(classes, "has-text-align-"+slugifySupport(ta))
				}
			}
			if border, ok := style["border"].(map[string]any); ok {
				if color := borderScalar(border["color"]); color != "" {
					classes = appendUnique(classes, "has-border-color")
					if slug := presetColorSlug(color); slug != "" {
						classes = appendUnique(classes, "has-"+slug+"-border-color")
					}
				}
			}
			// Background image/gradient support (P5.33 / wp_render_background_support).
			if bg, ok := style["background"].(map[string]any); ok && bg != nil {
				applyBackgroundDefaults(bg)
				if backgroundHasValue(bg) {
					classes = appendUnique(classes, "has-background")
				}
			}
			// Dimensions support (P5.34 / wp_render_dimensions_support).
			if dims, ok := style["dimensions"].(map[string]any); ok && dims != nil {
				applyDimensionsAspectRatio(dims, attrs)
				if isExplicitAspectRatio(dims["aspectRatio"]) {
					classes = appendUnique(classes, "has-aspect-ratio")
				}
			}
			styles = mergeCSSDecls(styles, CompileStyle(style))
			// Duotone class + compose filter: with any CSS filter functions (P5.22 / P5.35).
			if filterID, filterVal := duotoneFromStyle(style); filterID != "" {
				classes = appendUnique(classes, filterID)
				if filterVal != "" {
					styles = mergeFilterDecl(styles, filterVal)
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
		// Top-level minHeight attribute (cover/group etc., P5.34).
		// Skip when an explicit aspect-ratio is active (WP unsets min-height then).
		if mh := borderScalar(attrs["minHeight"]); mh != "" {
			skipMinH := false
			if style, ok := attrs["style"].(map[string]any); ok {
				if dims, ok := style["dimensions"].(map[string]any); ok {
					skipMinH = isExplicitAspectRatio(dims["aspectRatio"])
				}
			}
			if !skipMinH {
				styles = mergeCSSDecls(styles, "min-height:"+cssValue(mh))
			}
		}
		// Top-level shadow attribute (registered by wp_register_shadow_support).
		if shadow, ok := attrs["shadow"].(string); ok && shadow != "" {
			styles = mergeCSSDecls(styles, "box-shadow:"+cssValue(shadow))
		}

		// Aria-label block support (P5.36 / wp_apply_aria_label_support).
		if al, ok := attrs["ariaLabel"].(string); ok && strings.TrimSpace(al) != "" {
			rootAttrs["aria-label"] = strings.TrimSpace(al)
		}

		// Elements block support: link/button colors (P5.32).
		if elClasses, elCSS := elementsSupport(attrs); len(elClasses) > 0 || elCSS != "" {
			for _, c := range elClasses {
				classes = appendUnique(classes, c)
			}
			elementsCSS = elCSS
		}
	}
	classes = append(classes, layoutClasses(blockName, attrs)...)
	if layoutCSS := layoutInlineStyles(blockName, attrs); layoutCSS != "" {
		styles = mergeCSSDecls(styles, layoutCSS)
	}
	return classes, styles, elementsCSS, rootAttrs
}

// applyBackgroundDefaults mirrors wp_render_background_support size/position defaults (P5.33).
func applyBackgroundDefaults(bg map[string]any) {
	if bg == nil {
		return
	}
	if backgroundImageCSS(bg["backgroundImage"]) == "" {
		return
	}
	size := borderScalar(bg["backgroundSize"])
	if size == "" {
		bg["backgroundSize"] = "cover"
		size = "cover"
	}
	if size == "contain" && borderScalar(bg["backgroundPosition"]) == "" {
		bg["backgroundPosition"] = "50% 50%"
	}
}

func backgroundHasValue(bg map[string]any) bool {
	if bg == nil {
		return false
	}
	if backgroundImageCSS(bg["backgroundImage"]) != "" {
		return true
	}
	return borderScalar(bg["gradient"]) != ""
}

// applyDimensionsAspectRatio mirrors wp_render_dimensions_support conflict rules (P5.34).
// Explicit aspect-ratio unsets minHeight/height; presence of minHeight unsets aspect-ratio.
func applyDimensionsAspectRatio(dims map[string]any, attrs map[string]any) {
	if dims == nil {
		return
	}
	if isExplicitAspectRatio(dims["aspectRatio"]) {
		dims["minHeight"] = "unset"
		dims["height"] = "unset"
		return
	}
	hasMinHeight := borderScalar(dims["minHeight"]) != ""
	if !hasMinHeight && attrs != nil {
		hasMinHeight = borderScalar(attrs["minHeight"]) != ""
	}
	if hasMinHeight {
		dims["aspectRatio"] = "unset"
	}
}

func isExplicitAspectRatio(v any) bool {
	s := strings.ToLower(strings.TrimSpace(borderScalar(v)))
	return s != "" && s != "auto" && s != "unset"
}

// elementsSupport mirrors wp-includes/block-supports/elements.php for link/button colors (P5.32).
func elementsSupport(attrs map[string]any) (classes []string, css string) {
	if attrs == nil {
		return nil, ""
	}
	style, ok := attrs["style"].(map[string]any)
	if !ok || style == nil {
		return nil, ""
	}
	elements, ok := style["elements"].(map[string]any)
	if !ok || elements == nil {
		return nil, ""
	}

	linkStyle, _ := elements["link"].(map[string]any)
	buttonStyle, _ := elements["button"].(map[string]any)
	needClass := elementHasColor(linkStyle) || elementHasHoverColor(linkStyle) || elementHasColor(buttonStyle)
	if !needClass {
		return nil, ""
	}

	n := atomic.AddUint64(&elementsSeq, 1)
	className := fmt.Sprintf("wp-elements-%d", n)
	classes = append(classes, className)

	if elementHasColor(linkStyle) {
		classes = append(classes, "has-link-color")
	}

	var rules []string
	if linkStyle != nil {
		sel := fmt.Sprintf(".%s a:where(:not(.wp-element-button))", className)
		if rule := CompileCSS(sel, elementColorStyle(linkStyle)); rule != "" {
			rules = append(rules, rule)
		}
		if hover, ok := linkStyle[":hover"].(map[string]any); ok && elementHasColor(hover) {
			hoverSel := fmt.Sprintf(".%s a:where(:not(.wp-element-button)):hover", className)
			if rule := CompileCSS(hoverSel, elementColorStyle(hover)); rule != "" {
				rules = append(rules, rule)
			}
		}
	}
	if buttonStyle != nil && elementHasColor(buttonStyle) {
		sel := fmt.Sprintf(".%s .wp-element-button, .%s .wp-block-button__link", className, className)
		if rule := CompileCSS(sel, elementColorStyle(buttonStyle)); rule != "" {
			rules = append(rules, rule)
		}
	}
	return classes, strings.Join(rules, "")
}

func elementHasColor(el map[string]any) bool {
	if el == nil {
		return false
	}
	color, ok := el["color"].(map[string]any)
	if !ok || color == nil {
		return false
	}
	for _, key := range []string{"text", "background", "gradient"} {
		if v, ok := color[key].(string); ok && strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

func elementHasHoverColor(el map[string]any) bool {
	if el == nil {
		return false
	}
	hover, _ := el[":hover"].(map[string]any)
	return elementHasColor(hover)
}

// elementColorStyle extracts the color subtree for style-engine compilation.
func elementColorStyle(el map[string]any) map[string]any {
	if el == nil {
		return nil
	}
	color, ok := el["color"].(map[string]any)
	if !ok || color == nil {
		return nil
	}
	return map[string]any{"color": color}
}

// presetColorSlug extracts a color slug from var:preset|color|slug or CSS var form.
func presetColorSlug(attr string) string {
	attr = strings.TrimSpace(attr)
	if strings.HasPrefix(attr, "var:preset|color|") {
		return slugifySupport(strings.TrimPrefix(attr, "var:preset|color|"))
	}
	const prefix = "var(--wp--preset--color--"
	if strings.HasPrefix(attr, prefix) && strings.HasSuffix(attr, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(attr, prefix), ")")
		return slugifySupport(inner)
	}
	return ""
}

// duotoneFromStyle extracts WP duotone support (P5.22): class wp-duotone-{slug}
// plus the CSS filter *value* (no "filter:" prefix) for preset / unset values.
// Attr paths: style.color.duotone (legacy) and style.filter.duotone (current).
func duotoneFromStyle(style map[string]any) (filterID, filterValue string) {
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
			filterValue = "var(--wp--preset--duotone--" + slug + ")"
			return filterID, filterValue
		}
		// CSS values like "unset".
		slug := slugifySupport(v)
		if slug == "" {
			slug = "custom"
		}
		return "wp-duotone-" + slug, html.EscapeString(v)
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

// mergeFilterDecl prepends duotoneValue onto an existing filter: declaration
// (or adds filter:duotoneValue). CSS filter functions compose as a space list.
func mergeFilterDecl(existing, duotoneValue string) string {
	duotoneValue = strings.TrimSpace(duotoneValue)
	if duotoneValue == "" {
		return existing
	}
	existing = strings.TrimSpace(existing)
	var before []string
	var filterVal string
	for _, part := range strings.Split(existing, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.Index(part, ":"); i > 0 {
			prop := strings.ToLower(strings.TrimSpace(part[:i]))
			if prop == "filter" {
				filterVal = strings.TrimSpace(part[i+1:])
				continue
			}
		}
		before = append(before, part)
	}
	composed := duotoneValue
	if filterVal != "" {
		composed = duotoneValue + " " + filterVal
	}
	out := strings.Join(before, ";")
	if out != "" && !strings.HasSuffix(out, ";") {
		out += ";"
	}
	return out + "filter:" + composed
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

// addRootAttr sets or replaces a named attribute on the first HTML opening tag.
func addRootAttr(htmlFragment, name, value string) string {
	if htmlFragment == "" || name == "" {
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
	escVal := html.EscapeString(value)
	lower := strings.ToLower(open)
	needle := strings.ToLower(name) + `="`
	if idx := strings.Index(lower, needle); idx >= 0 {
		valStart := idx + len(needle)
		valEnd := strings.Index(open[valStart:], `"`)
		if valEnd >= 0 {
			valEnd += valStart
			open = open[:valStart] + escVal + open[valEnd:]
		}
	} else {
		open = open + fmt.Sprintf(` %s="%s"`, name, escVal)
	}
	suffix := ">"
	if selfClose {
		suffix = " />"
	}
	return htmlFragment[:i+1] + open + suffix + rest[end+1:]
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
