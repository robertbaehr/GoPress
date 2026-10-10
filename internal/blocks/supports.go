package blocks

import (
	"fmt"
	"html"
	"strings"
)

// applySupports injects basic color/spacing/fontSize classes and styles from
// block attributes onto the rendered root element (P5.10).
func applySupports(htmlFragment string, attrs map[string]any) string {
	if htmlFragment == "" || attrs == nil {
		return htmlFragment
	}
	classes, styles := supportsFromAttrs(attrs)
	out := htmlFragment
	for _, c := range classes {
		out = addRootClass(out, c)
	}
	if styles != "" {
		out = addRootStyle(out, styles)
	}
	return out
}

func supportsFromAttrs(attrs map[string]any) (classes []string, styles string) {
	if cn, ok := attrs["className"].(string); ok && cn != "" {
		classes = append(classes, strings.Fields(cn)...)
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

	var styleParts []string
	if style, ok := attrs["style"].(map[string]any); ok {
		if color, ok := style["color"].(map[string]any); ok {
			if bg, ok := color["background"].(string); ok && bg != "" {
				styleParts = append(styleParts, "background-color:"+cssValue(bg))
				classes = appendUnique(classes, "has-background")
			}
			if tx, ok := color["text"].(string); ok && tx != "" {
				styleParts = append(styleParts, "color:"+cssValue(tx))
				classes = appendUnique(classes, "has-text-color")
			}
		}
		if spacing, ok := style["spacing"].(map[string]any); ok {
			styleParts = append(styleParts, spacingStyles("padding", spacing["padding"])...)
			styleParts = append(styleParts, spacingStyles("margin", spacing["margin"])...)
		}
	}
	return classes, strings.Join(styleParts, ";")
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
		merged := existing
		if merged != "" && !strings.HasSuffix(merged, ";") {
			merged += ";"
		}
		merged += style
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
