package blocks

import (
	"html"
	"regexp"
	"strings"
)

var reGroupTag = regexp.MustCompile(`(?i)wp-block-group\b`)

// renderGroup renders core/group (P5.58 / P5.107 / P5.150).
// Prefers saved markup; synthesizes a layout wrapper from inner blocks when
// missing; honors tagName; applies layout/align/className/anchor polish on
// saved and synthesized output.
// P5.150: coerce align to wide|full; gradient preset + fontSize + ariaLabel polish.
// Color/typography/spacing/border land via applySupports on the caller (sanitized attrs).
func (e *Env) renderGroup(block Block) string {
	e = e.ensure()
	attrs := sanitizeGroupBlockAttrs(block.Attrs)
	block.Attrs = attrs
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" || !reGroupTag.MatchString(out) {
		out = synthesizeGroupBlock(e, block)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return applyGroupAttrs(out, attrs)
}

// sanitizeGroupBlockAttrs coerces core/group supports.align to the block.json enum (P5.150).
// Allowed: wide, full. Other align values are dropped before synthesize/polish/supports.
func sanitizeGroupBlockAttrs(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	align := strings.TrimSpace(attrString(out, "align"))
	switch align {
	case "", "wide", "full":
		// keep
	default:
		delete(out, "align")
	}
	return out
}

func synthesizeGroupBlock(e *Env, block Block) string {
	inner := e.renderInnerBlocks(block)
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	tag := groupTagName(block.Attrs)
	class := groupClassFromAttrs(block.Attrs)
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	if anchor := strings.TrimSpace(attrString(block.Attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` class="`)
	b.WriteString(class)
	b.WriteString(`">`)
	b.WriteString(inner)
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">")
	return b.String()
}

func groupTagName(attrs map[string]any) string {
	tag := strings.ToLower(strings.TrimSpace(attrString(attrs, "tagName")))
	if isSafeHTMLTag(tag) {
		return tag
	}
	return "div"
}

func groupClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-group"
	info := layoutInfoFromAttrs("core/group", attrs)
	slug := info.Type
	if slug == "" || slug == "default" {
		slug = "flow"
	}
	if slug == "constrained" {
		class += " has-global-padding"
	}
	if info.Orientation != "" {
		class += " is-" + sanitizeClassToken(info.Orientation)
	}
	if info.Justify != "" {
		class += " is-content-justification-" + sanitizeClassToken(info.Justify)
	}
	if info.FlexWrap == "nowrap" {
		class += " is-nowrap"
	}
	class += " is-layout-" + slug
	class += " wp-block-group-is-layout-" + slug
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	// P5.150: typography.fontSize + color.gradients presets on synthesize.
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
	}
	return class
}

func applyGroupAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	// When layout attrs request flex/grid/constrained, ensure stem layout classes.
	info := layoutInfoFromAttrs("core/group", attrs)
	slug := info.Type
	if slug == "default" || slug == "" {
		slug = "flow"
	}
	if info.FromAttrs {
		cls := "is-layout-" + slug
		if !strings.Contains(out, cls) {
			out = addRootClass(out, cls)
		}
		stem := "wp-block-group-is-layout-" + slug
		if !strings.Contains(out, stem) {
			out = addRootClass(out, stem)
		}
		if slug == "constrained" && !strings.Contains(out, "has-global-padding") {
			out = addRootClass(out, "has-global-padding")
		}
		if info.Orientation != "" {
			out = addRootClass(out, "is-"+sanitizeClassToken(info.Orientation))
		}
		if info.Justify != "" {
			out = addRootClass(out, "is-content-justification-"+sanitizeClassToken(info.Justify))
		}
		if info.FlexWrap == "nowrap" {
			out = addRootClass(out, "is-nowrap")
		}
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		cls := "align" + sanitizeClassToken(align)
		if !strings.Contains(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	// P5.107 / P5.150: className / anchor / fontSize / gradient / ariaLabel polish.
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	// P5.150: typography.fontSize preset class (also via applySupports).
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
	}
	// P5.150: color.gradients preset → has-{slug}-gradient-background.
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
	}
	// P5.150: supports.ariaLabel → aria-label on root.
	if al := strings.TrimSpace(attrString(attrs, "ariaLabel")); al != "" {
		out = ensureRootAttribute(out, "aria-label", al)
	}
	// Retag root element when tagName differs from saved markup.
	tag := groupTagName(attrs)
	if tag != "div" {
		out = retagRootElement(out, tag)
	}
	return out
}

// retagRootElement swaps the first opening/closing tag name when safe.
func retagRootElement(htmlFragment, tag string) string {
	htmlFragment = strings.TrimSpace(htmlFragment)
	if htmlFragment == "" || !strings.HasPrefix(htmlFragment, "<") {
		return htmlFragment
	}
	// Find first tag name.
	end := strings.IndexAny(htmlFragment, " \t\n\r/>")
	if end < 2 {
		return htmlFragment
	}
	oldOpen := htmlFragment[1:end]
	if strings.EqualFold(oldOpen, tag) {
		return htmlFragment
	}
	if !isSafeHTMLTag(strings.ToLower(oldOpen)) {
		return htmlFragment
	}
	out := "<" + tag + htmlFragment[end:]
	closeOld := "</" + oldOpen + ">"
	closeNew := "</" + tag + ">"
	if i := strings.LastIndex(strings.ToLower(out), strings.ToLower(closeOld)); i >= 0 {
		out = out[:i] + closeNew + out[i+len(closeOld):]
	}
	return out
}
