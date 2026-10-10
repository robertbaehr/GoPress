package blocks

import (
	"html"
	"regexp"
	"strings"
)

var rePullquoteTag = regexp.MustCompile(`(?i)wp-block-pullquote\b`)

// renderPullquote renders core/pullquote (P5.61).
// Prefers saved markup; synthesizes figure>blockquote from value/citation;
// honors align and className.
func (e *Env) renderPullquote(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && rePullquoteTag.MatchString(out) {
		return out
	}
	return synthesizePullquoteBlock(block.Attrs)
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
	return class
}
