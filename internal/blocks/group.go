package blocks

import (
	"regexp"
	"strings"
)

var reGroupTag = regexp.MustCompile(`(?i)wp-block-group\b`)

// renderGroup renders core/group (P5.58).
// Prefers saved markup; synthesizes a layout wrapper from inner blocks when
// missing; honors tagName; applies layout/align polish only on synthesized output
// so saved markup class order stays WP-parity for the front HTML corpus.
func (e *Env) renderGroup(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	synthesized := false
	if strings.TrimSpace(out) == "" || !reGroupTag.MatchString(out) {
		out = synthesizeGroupBlock(e, block)
		synthesized = true
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	if synthesized {
		return applyGroupAttrs(out, block.Attrs)
	}
	// Saved markup: only retag when tagName requests a non-div element.
	tag := groupTagName(block.Attrs)
	if tag != "div" {
		out = retagRootElement(out, tag)
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
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		cls := "align" + sanitizeClassToken(align)
		if !strings.Contains(out, cls) {
			out = addRootClass(out, cls)
		}
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
