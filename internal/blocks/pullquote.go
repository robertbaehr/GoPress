package blocks

import (
	"html"
	"regexp"
	"strings"
)

var rePullquoteTag = regexp.MustCompile(`(?i)wp-block-pullquote\b`)

// renderPullquote renders core/pullquote (P5.61 / P5.159).
// Prefers saved markup; synthesizes figure>blockquote from value/citation;
// honors align and className.
// P5.159: coerce align left|right|wide|full; fontSize/gradient/anchor polish.
func (e *Env) renderPullquote(block Block) string {
	e = e.ensure()
	attrs := sanitizePullquoteBlockAttrs(block.Attrs)
	block.Attrs = attrs
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && rePullquoteTag.MatchString(out) {
		return applyPullquoteAttrs(out, attrs)
	}
	return applyPullquoteAttrs(synthesizePullquoteBlock(attrs), attrs)
}

// sanitizePullquoteBlockAttrs coerces core/pullquote supports.align to block.json (P5.159).
// Allowed: left, right, wide, full. Other values (incl. center/none) are dropped.
func sanitizePullquoteBlockAttrs(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	align := strings.TrimSpace(attrString(out, "align"))
	switch align {
	case "", "left", "right", "wide", "full":
		// keep
	default:
		delete(out, "align")
	}
	return out
}

func synthesizePullquoteBlock(attrs map[string]any) string {
	value := strings.TrimSpace(attrString(attrs, "value"))
	citation := strings.TrimSpace(attrString(attrs, "citation"))
	class := pullquoteClassFromAttrs(attrs)

	var b strings.Builder
	b.WriteString(`<figure class="`)
	b.WriteString(class)
	b.WriteString(`"><blockquote>`)
	if value != "" {
		// Value may already contain <p>…</p>; wrap bare text.
		if strings.Contains(value, "<") {
			b.WriteString(value)
		} else {
			b.WriteString("<p>")
			b.WriteString(value)
			b.WriteString("</p>")
		}
	}
	if citation != "" {
		b.WriteString("<cite>")
		b.WriteString(citation)
		b.WriteString("</cite>")
	}
	b.WriteString("</blockquote></figure>")
	return b.String()
}

func pullquoteClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-pullquote"
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	// P5.159: typography.fontSize + color.gradients presets on synthesize.
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
	}
	return class
}

// applyPullquoteAttrs polishes fontSize/gradient/anchor on core/pullquote (P5.159).
func applyPullquoteAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if strings.TrimSpace(out) == "" {
		return out
	}
	out = addRootClass(out, "wp-block-pullquote")
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		cls := "align" + sanitizeClassToken(align)
		if !hasRootClass(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
	}
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	return out
}
