package blocks

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reColumnsTag = regexp.MustCompile(`(?i)wp-block-columns\b`)
	reColumnTag  = regexp.MustCompile(`(?i)wp-block-column\b`)
)

// renderColumns renders core/columns (P5.57).
// Prefers saved markup; synthesizes a flex wrapper from inner columns when
// missing; applies is-not-stacked-on-mobile and are-vertically-aligned-* from attrs.
func (e *Env) renderColumns(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" || !reColumnsTag.MatchString(out) {
		out = synthesizeColumnsBlock(e, block)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return applyColumnsAttrs(out, block.Attrs)
}

// renderColumn renders core/column (P5.57).
// Prefers saved markup; synthesizes from inner blocks; applies width flex-basis
// and is-vertically-aligned-* from attrs.
func (e *Env) renderColumn(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" || !reColumnTag.MatchString(out) {
		out = synthesizeColumnBlock(e, block)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return applyColumnAttrs(out, block.Attrs)
}

func synthesizeColumnsBlock(e *Env, block Block) string {
	inner := e.renderInnerBlocks(block)
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	class := columnsClassFromAttrs(block.Attrs)
	return `<div class="` + class + `">` + inner + `</div>`
}

func synthesizeColumnBlock(e *Env, block Block) string {
	inner := e.renderInnerBlocks(block)
	class := columnClassFromAttrs(block.Attrs)
	style := columnWidthStyle(block.Attrs)
	var b strings.Builder
	b.WriteString(`<div class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if style != "" {
		b.WriteString(` style="`)
		b.WriteString(html.EscapeString(style))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(inner)
	b.WriteString(`</div>`)
	return b.String()
}

func columnsClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-columns"
	if vAlign := strings.TrimSpace(attrString(attrs, "verticalAlignment")); vAlign != "" {
		class += " are-vertically-aligned-" + sanitizeClassToken(vAlign)
	}
	// Default isStackedOnMobile=true → no class; false → is-not-stacked-on-mobile.
	if attrs != nil {
		if _, ok := attrs["isStackedOnMobile"]; ok && !attrBool(attrs, "isStackedOnMobile") {
			class += " is-not-stacked-on-mobile"
		}
	}
	return class
}

func columnClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-column"
	if vAlign := strings.TrimSpace(attrString(attrs, "verticalAlignment")); vAlign != "" {
		class += " is-vertically-aligned-" + sanitizeClassToken(vAlign)
	}
	return class
}

func columnWidthStyle(attrs map[string]any) string {
	width := strings.TrimSpace(attrString(attrs, "width"))
	if width == "" {
		if n, ok := attrInt(attrs, "width"); ok && n > 0 {
			width = strconv.Itoa(n) + "%"
		} else if f, ok := attrFloat64(attrs, "width"); ok && f > 0 {
			width = formatFlexBasisPercent(f)
		}
	}
	if width == "" || !strings.ContainsAny(width, "0123456789") {
		return ""
	}
	basis := width
	if _, err := strconv.ParseFloat(width, 64); err == nil {
		basis = width + "%"
	} else if strings.HasSuffix(width, "%") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(width, "%"), 64); err == nil {
			basis = formatFlexBasisPercent(f)
		}
	}
	return "flex-basis:" + basis
}

func formatFlexBasisPercent(f float64) string {
	const multiplier = 1_000_000_000_000
	rounded := float64(int64(f*multiplier+0.5)) / multiplier
	return strconv.FormatFloat(rounded, 'f', -1, 64) + "%"
}

func attrFloat64(attrs map[string]any, key string) (float64, bool) {
	if attrs == nil {
		return 0, false
	}
	v, ok := attrs[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func applyColumnsAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if vAlign := strings.TrimSpace(attrString(attrs, "verticalAlignment")); vAlign != "" {
		cls := "are-vertically-aligned-" + sanitizeClassToken(vAlign)
		if !strings.Contains(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	if attrs != nil {
		if _, ok := attrs["isStackedOnMobile"]; ok && !attrBool(attrs, "isStackedOnMobile") {
			if !strings.Contains(out, "is-not-stacked-on-mobile") {
				out = addRootClass(out, "is-not-stacked-on-mobile")
			}
		}
	}
	return out
}

func applyColumnAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if vAlign := strings.TrimSpace(attrString(attrs, "verticalAlignment")); vAlign != "" {
		cls := "is-vertically-aligned-" + sanitizeClassToken(vAlign)
		if !strings.Contains(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	if style := columnWidthStyle(attrs); style != "" {
		if !strings.Contains(strings.ToLower(out), "flex-basis") {
			out = addRootStyle(out, style)
		}
	}
	return out
}
