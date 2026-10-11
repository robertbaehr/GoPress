package blocks

import (
	"html"
	"regexp"
	"strings"
)

var reSeparatorTag = regexp.MustCompile(`(?i)wp-block-separator\b`)

// renderSeparator renders core/separator (P5.60 / P5.112).
// Prefers saved markup; synthesizes hr/div with opacity + align + className.
// P5.112: anchor + className polish on saved and synthesized markup;
// background/gradient colors land via applySupports on the caller.
func (e *Env) renderSeparator(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reSeparatorTag.MatchString(out) {
		return applySeparatorAttrs(out, block.Attrs)
	}
	out = synthesizeSeparatorBlock(block.Attrs)
	return applySeparatorAttrs(out, block.Attrs)
}

func synthesizeSeparatorBlock(attrs map[string]any) string {
	tag := separatorTagName(attrs)
	class := separatorClassFromAttrs(attrs)
	var b strings.Builder
	b.WriteString(`<`)
	b.WriteString(tag)
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` class="`)
	b.WriteString(class)
	b.WriteString(`"/>`)
	return b.String()
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

// applySeparatorAttrs polishes className/align/opacity/anchor on separator roots (P5.112).
func applySeparatorAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if strings.TrimSpace(out) == "" {
		return out
	}
	out = addRootClass(out, "wp-block-separator")
	opacity := strings.TrimSpace(attrString(attrs, "opacity"))
	if opacity == "" {
		opacity = "alpha-channel"
	}
	switch opacity {
	case "css":
		if !strings.Contains(out, "has-css-opacity") {
			out = addRootClass(out, "has-css-opacity")
		}
	default:
		if !strings.Contains(out, "has-alpha-channel-opacity") && !strings.Contains(out, "has-css-opacity") {
			out = addRootClass(out, "has-alpha-channel-opacity")
		}
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		cls := "align" + sanitizeClassToken(align)
		if !strings.Contains(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	// Retag when tagName=div and saved/synth used hr (or vice versa).
	tag := separatorTagName(attrs)
	out = retagRootElement(out, tag)
	return out
}
