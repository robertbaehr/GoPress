package blocks

import "strings"

// renderHTML renders core/html (P5.68).
// Prefers saved inner markup; falls back to the content attribute (self-closing
// / dynamic forms). className support is disabled in block.json — no wrapper.
func (e *Env) renderHTML(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" {
		return out
	}
	if content, ok := block.Attrs["content"].(string); ok && content != "" {
		return content
	}
	return block.InnerHTML
}

// renderFreeform renders core/freeform / Classic (P5.68).
// Prefers the content attribute (source: raw), else saved/inner HTML.
func (e *Env) renderFreeform(block Block) string {
	e = e.ensure()
	if content, ok := block.Attrs["content"].(string); ok && content != "" {
		return content
	}
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" {
		return out
	}
	return block.InnerHTML
}
