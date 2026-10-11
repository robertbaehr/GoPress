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

// renderFile renders core/file (P5.52 / P5.114 / P5.156).
// Prefers saved markup; synthesizes from href/fileName when empty; when
// displayPreview is set, mirrors render_block_core_file interactive/object attrs.
// P5.114: className/align/anchor polish on the file wrapper.
// P5.156: coerce align left|center|right|wide|full; gradient polish.
func (e *Env) renderFile(block Block) string {
	e = e.ensure()
	attrs := sanitizeFileBlockAttrs(block.Attrs)
	block.Attrs = attrs
	out := e.renderSaved(block)
	if strings.TrimSpace(out) == "" {
		out = synthesizeFileBlock(attrs)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	if attrBool(attrs, "displayPreview") {
		out = applyFilePreviewAttrs(out, attrs)
	}
	out = applyMediaFigureAttrs(out, "wp-block-file", attrs)
	return applyFileAttrs(out, attrs)
}

// sanitizeFileBlockAttrs coerces core/file supports.align to the block.json set (P5.156).
// Allowed: left, center, right, wide, full (align: true). Other align values (incl. none) are dropped.
func sanitizeFileBlockAttrs(attrs map[string]any) map[string]any {
	if attrs == nil {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	align := strings.TrimSpace(attrString(out, "align"))
	switch align {
	case "", "left", "center", "right", "wide", "full":
		// keep
	case "none":
		delete(out, "align")
	default:
		delete(out, "align")
	}
	return out
}

// applyFileAttrs polishes gradient (and reinforces fontSize if present) on core/file (P5.156).
func applyFileAttrs(htmlFragment string, attrs map[string]any) string {
	out := htmlFragment
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
	}
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
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
	class := "wp-block-file"
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" && align != "none" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
	}
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	b.WriteString(`<div`)
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(` class="`)
	b.WriteString(class)
	b.WriteString(`">`)
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
