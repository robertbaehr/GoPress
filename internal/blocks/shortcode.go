package blocks

import "strings"

// renderShortcode renders core/shortcode (P5.80).
// Mirrors render_block_core_shortcode: wpautop on the shortcode text / inner HTML.
// Shortcode expansion happens later via the_content → DoShortcode.
func (e *Env) renderShortcode(block Block) string {
	e = e.ensure()
	content := strings.TrimSpace(attrString(block.Attrs, "text"))
	if content == "" {
		content = strings.TrimSpace(e.renderSaved(block))
	}
	if content == "" {
		content = strings.TrimSpace(block.InnerHTML)
	}
	if content == "" {
		return ""
	}
	return wpautopShortcode(content)
}

// wpautopShortcode is a minimal wpautop for shortcode block content.
func wpautopShortcode(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "<p>") || strings.Contains(raw, "<p ") {
		if !strings.HasSuffix(raw, "\n") {
			raw += "\n"
		}
		return raw
	}
	parts := strings.Split(raw, "\n")
	var b strings.Builder
	b.WriteString("<p>")
	for i, part := range parts {
		if i > 0 {
			b.WriteString("<br />\n")
		}
		b.WriteString(part)
	}
	b.WriteString("</p>\n")
	return b.String()
}
