package blocks

import (
	"html"
	"regexp"
	"strings"
)

var reQuoteTag = regexp.MustCompile(`(?i)wp-block-quote\b`)

// renderQuote renders core/quote (P5.61 / P5.103 / P5.146).
// Prefers saved markup; synthesizes blockquote from value/citation/inner blocks;
// always applies flow layout classes; honors textAlign, align, className, anchor.
// P5.146: coerce align enum (left|right|wide|full); gradient preset + fontSize polish.
// Color/typography/spacing land via applySupports on the caller (sanitized attrs).
func (e *Env) renderQuote(block Block) string {
	e = e.ensure()
	attrs := sanitizeQuoteBlockAttrs(block.Attrs)
	block.Attrs = attrs
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reQuoteTag.MatchString(out) {
		out = polishQuoteMarkup(out, attrs)
	} else {
		out = synthesizeQuoteBlock(e, block)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	out = addRootClass(out, "is-layout-flow")
	out = addRootClass(out, "wp-block-quote-is-layout-flow")
	return out
}

// sanitizeQuoteBlockAttrs coerces core/quote supports.align to the block.json enum (P5.146).
// Allowed: left, right, wide, full. Other align values are dropped before
// synthesize/polish/supports.
func sanitizeQuoteBlockAttrs(attrs map[string]any) map[string]any {
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

// polishQuoteMarkup ensures wp-block-quote + textAlign/align/className/anchor/gradient/fontSize
// on saved HTML (P5.103 / P5.146).
func polishQuoteMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-quote")
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		out = addRootClass(out, "has-text-align-"+sanitizeClassToken(ta))
	}
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
	// P5.146: typography.fontSize preset class (also via applySupports).
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
	}
	// P5.146: color.gradients preset → has-{slug}-gradient-background.
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
	}
	return out
}

func synthesizeQuoteBlock(e *Env, block Block) string {
	inner := ""
	if e != nil {
		inner = e.renderInnerBlocks(block)
	}
	if strings.TrimSpace(inner) == "" {
		if value := strings.TrimSpace(attrString(block.Attrs, "value")); value != "" {
			inner = value
		}
	}
	citation := strings.TrimSpace(attrString(block.Attrs, "citation"))
	if strings.TrimSpace(inner) == "" && citation == "" {
		// Empty quote still emits a blockquote shell (editor parity).
		inner = ""
	}
	class := quoteClassFromAttrs(block.Attrs)
	var b strings.Builder
	b.WriteString(`<blockquote class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(inner)
	if citation != "" {
		b.WriteString("<cite>")
		b.WriteString(citation)
		b.WriteString("</cite>")
	}
	b.WriteString("</blockquote>")
	return b.String()
}

func quoteClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-quote"
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		class += " has-text-align-" + sanitizeClassToken(ta)
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
	}
	return class
}
