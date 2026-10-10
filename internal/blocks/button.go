package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	reButtonsTag        = regexp.MustCompile(`(?i)wp-block-buttons\b`)
	reButtonWrapperOpen = regexp.MustCompile(`(?is)^(\s*<div\b)([^>]*\bwp-block-button\b[^>]*)(>)`)
	reButtonLinkOpen    = regexp.MustCompile(`(?is)<(a|button)\b[^>]*>`)
	reButtonEmptyInner  = regexp.MustCompile(`(?is)<(a|button)\b[^>]*>(?:\s|<!--.*?-->)*</(a|button)>`)
)

// renderButtons renders core/buttons (P5.56).
// Prefers saved markup; when the wrapper is missing, wraps rendered inner buttons.
func (e *Env) renderButtons(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeButtonsBlock(e, block)
	} else if !reButtonsTag.MatchString(out) {
		out = wrapButtonsInner(block.Attrs, out)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return out
}

// renderButton renders core/button (P5.56).
// Prefers saved markup; synthesizes from text/url attrs; empty buttons render
// nothing; applies width classes from style.dimensions.width.
func (e *Env) renderButton(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeButtonBlock(block.Attrs)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	if buttonContentEmpty(out) {
		return ""
	}
	return applyButtonWidth(out, block.Attrs)
}

func synthesizeButtonsBlock(e *Env, block Block) string {
	inner := e.renderInnerBlocks(block)
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	return wrapButtonsInner(block.Attrs, inner)
}

func wrapButtonsInner(attrs map[string]any, inner string) string {
	if strings.TrimSpace(inner) == "" {
		return ""
	}
	class := "wp-block-buttons is-layout-flex wp-block-buttons-is-layout-flex"
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + align
	}
	return `<div class="` + class + `">` + inner + `</div>`
}

func synthesizeButtonBlock(attrs map[string]any) string {
	text := strings.TrimSpace(attrString(attrs, "text"))
	if text == "" {
		return ""
	}
	tag := strings.ToLower(strings.TrimSpace(attrString(attrs, "tagName")))
	if tag != "button" {
		tag = "a"
	}
	url := strings.TrimSpace(attrString(attrs, "url"))
	target := strings.TrimSpace(attrString(attrs, "linkTarget"))
	rel := strings.TrimSpace(attrString(attrs, "rel"))
	title := strings.TrimSpace(attrString(attrs, "title"))

	var b strings.Builder
	b.WriteString(`<div class="wp-block-button"><`)
	b.WriteString(tag)
	b.WriteString(` class="wp-block-button__link wp-element-button"`)
	if tag == "a" && url != "" {
		b.WriteString(` href="`)
		b.WriteString(html.EscapeString(url))
		b.WriteString(`"`)
	}
	if tag == "button" {
		typ := strings.TrimSpace(attrString(attrs, "type"))
		if typ == "" {
			typ = "button"
		}
		b.WriteString(` type="`)
		b.WriteString(html.EscapeString(typ))
		b.WriteString(`"`)
	}
	if target != "" {
		b.WriteString(` target="`)
		b.WriteString(html.EscapeString(target))
		b.WriteString(`"`)
	}
	if rel != "" {
		b.WriteString(` rel="`)
		b.WriteString(html.EscapeString(rel))
		b.WriteString(`"`)
	}
	if title != "" {
		b.WriteString(` title="`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(text)
	b.WriteString(`</`)
	b.WriteString(tag)
	b.WriteString(`></div>`)
	return b.String()
}

// buttonContentEmpty mirrors render_block_core_button empty-button check.
func buttonContentEmpty(content string) bool {
	if !reButtonLinkOpen.MatchString(content) {
		return false
	}
	return reButtonEmptyInner.MatchString(content)
}

// applyButtonWidth mirrors WP render_block_core_button width handling (P5.56).
func applyButtonWidth(content string, attrs map[string]any) string {
	width := buttonWidthFromAttrs(attrs)
	if width == "" {
		return content
	}
	m := reButtonWrapperOpen.FindStringSubmatchIndex(content)
	if m == nil {
		return content
	}
	open := content[m[2]:m[3]]
	attrPart := content[m[4]:m[5]]
	close := content[m[6]:m[7]]
	rest := content[m[1]:]

	resolved := width
	isPreset := strings.HasPrefix(width, "var:preset|dimension|")
	slug := ""
	if isPreset {
		slug = strings.TrimPrefix(width, "var:preset|dimension|")
		resolved = width // keep as CSS var reference when synthesizing style
	}
	isPercentage := strings.HasSuffix(resolved, "%") && !isPreset

	attrPart = addHTMLClass(attrPart, "has-custom-width")
	existingStyle := htmlAttrValue(attrPart, "style")

	var widthStyle string
	if isPercentage {
		attrPart = addHTMLClass(attrPart, "wp-block-button__width")
		legacy := map[string]string{
			"25%":  "wp-block-button__width-25",
			"50%":  "wp-block-button__width-50",
			"75%":  "wp-block-button__width-75",
			"100%": "wp-block-button__width-100",
		}
		if cls, ok := legacy[resolved]; ok {
			attrPart = addHTMLClass(attrPart, cls)
		}
		numeric := strings.TrimSuffix(resolved, "%")
		widthStyle = fmt.Sprintf("--wp--block-button--width: %s;", numeric)
	} else if isPreset && slug != "" {
		cssVal := "var(--wp--preset--dimension--" + camelToKebab(slug) + ")"
		widthStyle = "width: " + cssVal + ";"
	} else {
		widthStyle = "width: " + width + ";"
	}

	newStyle := widthStyle
	if existingStyle != "" {
		newStyle = widthStyle + " " + existingStyle
	}
	attrPart = upsertHTMLAttr(attrPart, "style", newStyle)

	return open + attrPart + close + rest
}

func buttonWidthFromAttrs(attrs map[string]any) string {
	if attrs == nil {
		return ""
	}
	style, _ := attrs["style"].(map[string]any)
	if style == nil {
		return ""
	}
	dims, _ := style["dimensions"].(map[string]any)
	if dims == nil {
		return ""
	}
	return strings.TrimSpace(attrString(dims, "width"))
}

func addHTMLClass(attrPart, className string) string {
	className = strings.TrimSpace(className)
	if className == "" {
		return attrPart
	}
	re := regexp.MustCompile(`(?i)\bclass\s*=\s*("([^"]*)"|'([^']*)')`)
	m := re.FindStringSubmatchIndex(attrPart)
	if m == nil {
		return strings.TrimRight(attrPart, " ") + ` class="` + html.EscapeString(className) + `"`
	}
	var existing string
	if m[4] >= 0 {
		existing = attrPart[m[4]:m[5]]
	} else if m[6] >= 0 {
		existing = attrPart[m[6]:m[7]]
	}
	classes := strings.Fields(existing)
	for _, c := range classes {
		if c == className {
			return attrPart
		}
	}
	classes = append(classes, className)
	replacement := ` class="` + strings.Join(classes, " ") + `"`
	return attrPart[:m[0]] + replacement + attrPart[m[1]:]
}
