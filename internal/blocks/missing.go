package blocks

import "strings"

// renderMissing renders core/missing (P5.69).
// Unsupported / removed blocks preserve their original markup via attributes:
// prefer originalContent (source: raw), then originalUndelimitedContent,
// then saved / inner HTML. className support is disabled in block.json.
func (e *Env) renderMissing(block Block) string {
	e = e.ensure()
	if oc := strings.TrimSpace(attrString(block.Attrs, "originalContent")); oc != "" {
		return attrString(block.Attrs, "originalContent")
	}
	if oud := strings.TrimSpace(attrString(block.Attrs, "originalUndelimitedContent")); oud != "" {
		return attrString(block.Attrs, "originalUndelimitedContent")
	}
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" {
		return out
	}
	return block.InnerHTML
}
