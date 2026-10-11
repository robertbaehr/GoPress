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

// renderColumns renders core/columns (P5.57 / P5.106 / P5.149).
// Prefers saved markup; synthesizes a flex wrapper from inner columns when
// missing; applies is-not-stacked-on-mobile, are-vertically-aligned-*,
// align/className/anchor from attrs.
// P5.149: coerce align to wide|full; gradient preset + fontSize polish.
// Color/typography/spacing/border land via applySupports on the caller (sanitized attrs).
func (e *Env) renderColumns(block Block) string {
	e = e.ensure()
	attrs := sanitizeColumnsBlockAttrs(block.Attrs)
	block.Attrs = attrs
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" || !reColumnsTag.MatchString(out) {
		out = synthesizeColumnsBlock(e, block)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return applyColumnsAttrs(out, attrs)
}

// sanitizeColumnsBlockAttrs coerces core/columns supports.align to the block.json enum (P5.149).
// Allowed: wide, full. Other align values are dropped before synthesize/polish/supports.
func sanitizeColumnsBlockAttrs(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	align := strings.TrimSpace(attrString(out, "align"))
	switch align {
	case "", "wide", "full":
		// keep
	default:
		delete(out, "align")
	}
	return out
}

// renderColumn renders core/column (P5.57 / P5.106).
// Prefers saved markup; synthesizes from inner blocks; applies width flex-basis,
// is-vertically-aligned-*, className/anchor from attrs.
// Color/typography/spacing/border land via applySupports on the caller.
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
	var b strings.Builder
	b.WriteString(`<div`)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` class="`)
	b.WriteString(class)
	b.WriteString(`">`)
	b.WriteString(inner)
	b.WriteString(`</div>`)
	return b.String()
}

func synthesizeColumnBlock(e *Env, block Block) string {
	inner := e.renderInnerBlocks(block)
	class := columnClassFromAttrs(block.Attrs)
	style := columnWidthStyle(block.Attrs)
	var b strings.Builder
	b.WriteString(`<div`)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` class="`)
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
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	// P5.149: typography.fontSize + color.gradients presets on synthesize.
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
	}
	return class
}

func columnClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-column"
	if vAlign := strings.TrimSpace(attrString(attrs, "verticalAlignment")); vAlign != "" {
		class += " is-vertically-aligned-" + sanitizeClassToken(vAlign)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
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
	// P5.106 / P5.149: align / className / anchor / fontSize / gradient polish.
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		out = addRootClass(out, "align"+sanitizeClassToken(align))
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	// P5.149: typography.fontSize preset class (also via applySupports).
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
	}
	// P5.149: color.gradients preset → has-{slug}-gradient-background.
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
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
	// P5.106: className / anchor polish.
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	return out
}
