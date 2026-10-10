package blocks

import (
	"html"
	"regexp"
	"strings"
)

var reQuoteTag = regexp.MustCompile(`(?i)wp-block-quote\b`)

// renderQuote renders core/quote (P5.61).
// Prefers saved markup; synthesizes blockquote from value/citation/inner blocks;
// always applies flow layout classes; honors textAlign, align, className.
func (e *Env) renderQuote(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" || !reQuoteTag.MatchString(out) {
		out = synthesizeQuoteBlock(e, block)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	out = addRootClass(out, "is-layout-flow")
	out = addRootClass(out, "wp-block-quote-is-layout-flow")
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
	b.WriteString(`">`)
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
	return class
}
