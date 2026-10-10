package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var reSeparatorTag = regexp.MustCompile(`(?i)wp-block-separator\b`)

// renderSeparator renders core/separator (P5.60).
// Prefers saved markup; synthesizes hr/div with opacity + align + className.
func (e *Env) renderSeparator(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reSeparatorTag.MatchString(out) {
		return out
	}
	return synthesizeSeparatorBlock(block.Attrs)
}

func synthesizeSeparatorBlock(attrs map[string]any) string {
	tag := separatorTagName(attrs)
	class := separatorClassFromAttrs(attrs)
	return fmt.Sprintf(`<%s class="%s"/>`, tag, class)
}

func separatorTagName(attrs map[string]any) string {
	tag := strings.ToLower(strings.TrimSpace(attrString(attrs, "tagName")))
	if tag == "div" {
		return "div"
	}
	return "hr"
}

func separatorClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-separator"
	opacity := strings.TrimSpace(attrString(attrs, "opacity"))
	if opacity == "" {
		opacity = "alpha-channel"
	}
	switch opacity {
	case "css":
		class += " has-css-opacity"
	default:
		class += " has-alpha-channel-opacity"
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	return class
}
