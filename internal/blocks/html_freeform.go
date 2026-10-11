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
	if content, ok := freeformContentAttr(block); ok {
		return content
	}
	return block.InnerHTML
}

// renderFreeform renders core/freeform / Classic (P5.68 / P5.129).
// Prefers the content attribute (source: raw), else saved/inner HTML, else
// InnerContent string fragments. Nameless classic HTML uses the same path.
func (e *Env) renderFreeform(block Block) string {
	e = e.ensure()
	if content, ok := freeformContentAttr(block); ok {
		return content
	}
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" {
		return out
	}
	if joined := freeformInnerContentJoin(block); strings.TrimSpace(joined) != "" {
		return joined
	}
	return block.InnerHTML
}

// freeformContentAttr returns the Classic/HTML content attribute when it is a
// non-empty string (P5.129). Null / non-string / empty → not present.
func freeformContentAttr(block Block) (string, bool) {
	if block.Attrs == nil {
		return "", false
	}
	raw, ok := block.Attrs["content"]
	if !ok || raw == nil {
		return "", false
	}
	s, ok := raw.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// freeformInnerContentJoin concatenates string fragments from InnerContent when
// there are no nested blocks (classic HTML stored as content pieces).
func freeformInnerContentJoin(block Block) string {
	if len(block.InnerBlocks) > 0 || len(block.InnerContent) == 0 {
		return ""
	}
	var b strings.Builder
	for _, part := range block.InnerContent {
		if s, ok := part.(string); ok {
			b.WriteString(s)
		}
	}
	return b.String()
}
