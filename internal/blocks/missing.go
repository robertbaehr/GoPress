package blocks

import "strings"

// renderMissing renders core/missing (P5.69 / P5.130).
// Unsupported / removed blocks preserve their original markup via attributes:
// prefer originalContent (source: raw), then originalUndelimitedContent,
// then saved / InnerContent string fragments / InnerHTML.
// className support is disabled in block.json — no wrapper.
func (e *Env) renderMissing(block Block) string {
	e = e.ensure()
	if content, ok := missingContentAttr(block, "originalContent"); ok {
		return content
	}
	if content, ok := missingContentAttr(block, "originalUndelimitedContent"); ok {
		return content
	}
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" {
		return out
	}
	if joined := missingInnerContentJoin(block); strings.TrimSpace(joined) != "" {
		return joined
	}
	return block.InnerHTML
}

// missingContentAttr returns a recovery markup attribute when it is a non-empty
// string (P5.130). Null / non-string / empty / whitespace-only → not present.
func missingContentAttr(block Block, key string) (string, bool) {
	if block.Attrs == nil {
		return "", false
	}
	raw, ok := block.Attrs[key]
	if !ok || raw == nil {
		return "", false
	}
	s, ok := raw.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}

// missingInnerContentJoin concatenates string fragments from InnerContent when
// there are no nested blocks (P5.130).
func missingInnerContentJoin(block Block) string {
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
