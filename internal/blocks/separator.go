package blocks

import (
	"html"
	"regexp"
	"strings"
)

var reSeparatorTag = regexp.MustCompile(`(?i)wp-block-separator\b`)

// renderSeparator renders core/separator (P5.60 / P5.112 / P5.142).
// Prefers saved markup; synthesizes hr/div with opacity + align + className.
// P5.112: anchor + className polish on saved and synthesized markup;
// background/gradient colors land via applySupports on the caller.
// P5.142: tagName enum coerce; align=center; gradient preset classes;
// spacing.margin via applySupports.
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

// separatorTagName returns hr|div only (block.json enum); other values → hr (P5.142).
func separatorTagName(attrs map[string]any) string {
	tag := strings.ToLower(strings.TrimSpace(attrString(attrs, "tagName")))
	if tag == "div" {
		return "div"
	}
	return "hr"
}

func separatorClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-separator"
	switch separatorOpacity(attrs) {
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
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
	}
	return class
}

// separatorOpacity normalizes opacity to css|alpha-channel (block.json default) (P5.142).
func separatorOpacity(attrs map[string]any) string {
	opacity := strings.TrimSpace(attrString(attrs, "opacity"))
	if opacity == "css" {
		return "css"
	}
	return "alpha-channel"
}

// applySeparatorAttrs polishes className/align/opacity/anchor/gradient on separator roots (P5.112 / P5.142).
func applySeparatorAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if strings.TrimSpace(out) == "" {
		return out
	}
	out = addRootClass(out, "wp-block-separator")
	switch separatorOpacity(attrs) {
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
	// P5.142: color.gradients preset → has-{slug}-gradient-background.
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
	}
	// Retag when tagName=div and saved/synth used hr (or vice versa); invalid → hr.
	tag := separatorTagName(attrs)
	out = retagRootElement(out, tag)
	return out
}
