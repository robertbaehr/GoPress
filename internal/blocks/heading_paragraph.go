package blocks

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reHeadingTag   = regexp.MustCompile(`(?i)<h[1-6]\b`)
	reParagraphTag = regexp.MustCompile(`(?i)<p\b`)
)

// renderHeading renders core/heading (P5.64 / P5.116).
// Prefers saved markup and ensures wp-block-heading; synthesizes from content/level
// when markup is absent; honors textAlign, align, className, anchor.
// Color/typography/spacing land via applySupports on the caller.
func (e *Env) renderHeading(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reHeadingTag.MatchString(out) {
		return polishHeadingMarkup(out, block.Attrs)
	}
	return synthesizeHeadingBlock(block.Attrs)
}

// polishHeadingMarkup ensures wp-block-heading + textAlign/align/className/anchor (P5.116).
func polishHeadingMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-heading")
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		cls := "has-text-align-" + sanitizeClassToken(ta)
		if !hasRootClass(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		cls := "align" + sanitizeClassToken(align)
		if !hasRootClass(out, cls) {
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
	return out
}

func synthesizeHeadingBlock(attrs map[string]any) string {
	content := attrString(attrs, "content")
	level := 2
	if n, ok := attrInt(attrs, "level"); ok && n >= 1 && n <= 6 {
		level = n
	}
	tag := "h" + strconv.Itoa(level)
	class := "wp-block-heading"
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		class += " has-text-align-" + sanitizeClassToken(ta)
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	var b strings.Builder
	b.WriteString(`<`)
	b.WriteString(tag)
	b.WriteString(` class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(content)
	b.WriteString(`</`)
	b.WriteString(tag)
	b.WriteString(`>`)
	return b.String()
}

// renderParagraph renders core/paragraph (P5.64 / P5.116).
// Prefers saved markup and ensures wp-block-paragraph; synthesizes from content
// when markup is absent; honors textAlign, align, dropCap, className, anchor.
// Color/typography/spacing land via applySupports on the caller.
func (e *Env) renderParagraph(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reParagraphTag.MatchString(out) {
		return polishParagraphMarkup(out, block.Attrs)
	}
	return synthesizeParagraphBlock(block.Attrs)
}

// polishParagraphMarkup ensures wp-block-paragraph + dropCap/textAlign/align/className/anchor (P5.116).
func polishParagraphMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-paragraph")
	if attrBool(attrs, "dropCap") {
		if !hasRootClass(out, "has-drop-cap") {
			out = addRootClass(out, "has-drop-cap")
		}
	}
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		cls := "has-text-align-" + sanitizeClassToken(ta)
		if !hasRootClass(out, cls) {
			out = addRootClass(out, cls)
		}
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		cls := "align" + sanitizeClassToken(align)
		if !hasRootClass(out, cls) {
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
	return out
}

func synthesizeParagraphBlock(attrs map[string]any) string {
	content := attrString(attrs, "content")
	class := "wp-block-paragraph"
	if attrBool(attrs, "dropCap") {
		class += " has-drop-cap"
	}
	if ta := strings.TrimSpace(attrString(attrs, "textAlign")); ta != "" {
		class += " has-text-align-" + sanitizeClassToken(ta)
	}
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	var b strings.Builder
	b.WriteString(`<p class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(content)
	b.WriteString(`</p>`)
	return b.String()
}
