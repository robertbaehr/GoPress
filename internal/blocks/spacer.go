package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var reSpacerTag = regexp.MustCompile(`(?i)wp-block-spacer\b`)

// renderSpacer renders core/spacer (P5.59).
// Prefers saved markup; synthesizes height/width (default height 100px);
// normalizes unitless numeric lengths to px; honors className.
func (e *Env) renderSpacer(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reSpacerTag.MatchString(out) {
		return out
	}
	return synthesizeSpacerBlock(block.Attrs)
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
	return fmt.Sprintf(`<div style="%s" aria-hidden="true" class="%s"></div>`, style, class)
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
