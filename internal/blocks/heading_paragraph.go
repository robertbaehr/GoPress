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

// renderHeading renders core/heading (P5.64).
// Prefers saved markup and ensures wp-block-heading; synthesizes from content/level
// when markup is absent; honors textAlign, align, className.
func (e *Env) renderHeading(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reHeadingTag.MatchString(out) {
		return addRootClass(out, "wp-block-heading")
	}
	return synthesizeHeadingBlock(block.Attrs)
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
	return `<` + tag + ` class="` + class + `">` + content + `</` + tag + `>`
}

// renderParagraph renders core/paragraph (P5.64).
// Prefers saved markup and ensures wp-block-paragraph; synthesizes from content
// when markup is absent; honors textAlign, align, dropCap, className.
func (e *Env) renderParagraph(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reParagraphTag.MatchString(out) {
		return addRootClass(out, "wp-block-paragraph")
	}
	return synthesizeParagraphBlock(block.Attrs)
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
	return `<p class="` + class + `">` + content + `</p>`
}
