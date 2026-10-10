package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	reFileDivOpen    = regexp.MustCompile(`(?is)^(\s*<div\b)([^>]*)(>)`)
	reFileObjectOpen = regexp.MustCompile(`(?is)<object\b([^>]*)>`)
)

// renderFile renders core/file (P5.52).
// Prefers saved markup; synthesizes from href/fileName when empty; when
// displayPreview is set, mirrors render_block_core_file interactive/object attrs.
func (e *Env) renderFile(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeFileBlock(block.Attrs)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	if attrBool(block.Attrs, "displayPreview") {
		out = applyFilePreviewAttrs(out, block.Attrs)
	}
	return out
}

func synthesizeFileBlock(attrs map[string]any) string {
	href := strings.TrimSpace(attrString(attrs, "href"))
	if href == "" {
		href = strings.TrimSpace(attrString(attrs, "textLinkHref"))
	}
	if href == "" {
		return ""
	}
	fileName := strings.TrimSpace(attrString(attrs, "fileName"))
	if fileName == "" {
		fileName = href
	}
	showDownload := true
	if attrs != nil {
		if _, ok := attrs["showDownloadButton"]; ok {
			showDownload = attrBool(attrs, "showDownloadButton")
		}
	}
	btnText := strings.TrimSpace(attrString(attrs, "downloadButtonText"))
	if btnText == "" {
		btnText = "Download"
	}
	displayPreview := attrBool(attrs, "displayPreview")
	previewHeight := 600
	if h, ok := attrInt(attrs, "previewHeight"); ok && h > 0 {
		previewHeight = h
	}

	escHref := html.EscapeString(href)
	escName := html.EscapeString(fileName)
	var b strings.Builder
	b.WriteString(`<div class="wp-block-file">`)
	if displayPreview {
		b.WriteString(`<object class="wp-block-file__embed" data="`)
		b.WriteString(escHref)
		b.WriteString(`" type="application/pdf" style="height:`)
		b.WriteString(fmt.Sprintf("%dpx", previewHeight))
		b.WriteString(`" aria-label="PDF embed"></object>`)
	}
	b.WriteString(`<a href="`)
	b.WriteString(escHref)
	b.WriteString(`">`)
	b.WriteString(escName)
	b.WriteString(`</a>`)
	if showDownload {
		b.WriteString(`<a href="`)
		b.WriteString(escHref)
		b.WriteString(`" class="wp-block-file__button" download>`)
		b.WriteString(html.EscapeString(btnText))
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// applyFilePreviewAttrs mirrors WP render_block_core_file Tag Processor updates.
func applyFilePreviewAttrs(content string, attrs map[string]any) string {
	m := reFileDivOpen.FindStringSubmatchIndex(content)
	if m == nil {
		return content
	}
	open := content[m[2]:m[3]]
	attrPart := content[m[4]:m[5]]
	close := content[m[6]:m[7]]
	rest := content[m[1]:]

	if !strings.Contains(strings.ToLower(attrPart), "data-wp-interactive") {
		attrPart = strings.TrimRight(attrPart, " ") + ` data-wp-interactive="core/file"`
	}

	var b strings.Builder
	b.WriteString(open)
	b.WriteString(attrPart)
	b.WriteString(close)

	obj := reFileObjectOpen.FindStringSubmatchIndex(rest)
	if obj == nil {
		b.WriteString(rest)
		return b.String()
	}
	before := rest[:obj[0]]
	objAttrs := rest[obj[2]:obj[3]]
	after := rest[obj[1]:]

	filename := objectAriaLabel(objAttrs)
	label := "PDF embed"
	if filename != "" && !strings.EqualFold(filename, "PDF embed") {
		label = "Embed of " + filename + "."
	}
	objAttrs = upsertHTMLAttr(objAttrs, "data-wp-bind--hidden", "!state.hasPdfPreview")
	objAttrs = upsertHTMLAttr(objAttrs, "hidden", "true")
	objAttrs = upsertHTMLAttr(objAttrs, "aria-label", label)
	_ = attrs

	b.WriteString(before)
	b.WriteString("<object")
	b.WriteString(objAttrs)
	b.WriteString(">")
	b.WriteString(after)
	return b.String()
}

func objectAriaLabel(attrPart string) string {
	re := regexp.MustCompile(`(?i)\baria-label\s*=\s*"([^"]*)"`)
	m := re.FindStringSubmatch(attrPart)
	if len(m) < 2 {
		re = regexp.MustCompile(`(?i)\baria-label\s*=\s*'([^']*)'`)
		m = re.FindStringSubmatch(attrPart)
	}
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func upsertHTMLAttr(attrPart, name, value string) string {
	re := regexp.MustCompile(`(?i)\s` + regexp.QuoteMeta(name) + `\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	replacement := ` ` + name + `="` + html.EscapeString(value) + `"`
	if re.MatchString(attrPart) {
		return re.ReplaceAllString(attrPart, replacement)
	}
	return strings.TrimRight(attrPart, " ") + replacement
}
