package blocks

import (
	"html"
	"regexp"
	"strings"
)

var reSpacerTag = regexp.MustCompile(`(?i)wp-block-spacer\b`)

// renderSpacer renders core/spacer (P5.59 / P5.112).
// Prefers saved markup; synthesizes height/width (default height 100px);
// normalizes unitless numeric lengths to px; honors className.
// P5.112: anchor + className polish on saved and synthesized markup.
func (e *Env) renderSpacer(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reSpacerTag.MatchString(out) {
		return applySpacerAttrs(out, block.Attrs)
	}
	out = synthesizeSpacerBlock(block.Attrs)
	return applySpacerAttrs(out, block.Attrs)
}

func synthesizeSpacerBlock(attrs map[string]any) string {
	height := attrLength(attrs["height"])
	width := attrLength(attrs["width"])
	// Horizontal flex spacers often set width only; keep height omitted then.
	if height == "" && width == "" {
		height = "100px"
	}
	height = normalizeCSSLength(height)
	width = normalizeCSSLength(width)

	var styleParts []string
	if height != "" {
		styleParts = append(styleParts, "height:"+html.EscapeString(height))
	}
	if width != "" {
		styleParts = append(styleParts, "width:"+html.EscapeString(width))
	}
	style := strings.Join(styleParts, ";")

	class := "wp-block-spacer"
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	var b strings.Builder
	b.WriteString(`<div`)
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` style="`)
	b.WriteString(style)
	b.WriteString(`" aria-hidden="true" class="`)
	b.WriteString(class)
	b.WriteString(`"></div>`)
	return b.String()
}

// normalizeCSSLength appends px to bare numeric lengths (WP spacer attrs).
func normalizeCSSLength(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	// Already has a unit or is a CSS function/keyword.
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '%' || r == '(' {
			return v
		}
	}
	// Digits / decimal / sign only → treat as px.
	return v + "px"
}

// applySpacerAttrs polishes className/anchor on spacer roots (P5.112).
func applySpacerAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if strings.TrimSpace(out) == "" {
		return out
	}
	out = addRootClass(out, "wp-block-spacer")
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
