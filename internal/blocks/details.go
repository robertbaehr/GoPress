package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	reDetailsTag = regexp.MustCompile(`(?i)<details\b`)
	reImgTagOpen = regexp.MustCompile(`(?i)<img\b([^>]*)>`)
)

// renderDetails renders core/details (P5.67).
// Prefers saved markup; ensures wp-block-details + align/className/name;
// honors showContent → open; synthesizes from summary + inner blocks;
// sets fetchpriority="low" on nested imgs when collapsed (showContent false).
func (e *Env) renderDetails(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reDetailsTag.MatchString(out) {
		out = polishDetailsMarkup(out, block.Attrs)
	} else {
		out = synthesizeDetailsBlock(e, block)
	}
	if strings.TrimSpace(out) == "" {
		return ""
	}
	if !attrBool(block.Attrs, "showContent") {
		out = detailsSetImgFetchPriorityLow(out)
	}
	return out
}

func polishDetailsMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-details")
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		out = addRootClass(out, "align"+sanitizeClassToken(align))
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if name := strings.TrimSpace(attrString(attrs, "name")); name != "" {
		out = ensureRootAttribute(out, "name", name)
	}
	if attrBool(attrs, "showContent") {
		out = ensureRootBooleanAttribute(out, "open")
	}
	return out
}

func synthesizeDetailsBlock(e *Env, block Block) string {
	summary := strings.TrimSpace(attrString(block.Attrs, "summary"))
	inner := ""
	if e != nil {
		inner = e.renderInnerBlocks(block)
	}
	if summary == "" && strings.TrimSpace(inner) == "" {
		return ""
	}
	class := "wp-block-details"
	if align := strings.TrimSpace(attrString(block.Attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(block.Attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	var b strings.Builder
	b.WriteString(`<details class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if name := strings.TrimSpace(attrString(block.Attrs, "name")); name != "" {
		b.WriteString(` name="`)
		b.WriteString(html.EscapeString(name))
		b.WriteString(`"`)
	}
	if attrBool(block.Attrs, "showContent") {
		b.WriteString(" open")
	}
	b.WriteString(`><summary>`)
	b.WriteString(summary)
	b.WriteString(`</summary>`)
	b.WriteString(inner)
	b.WriteString(`</details>`)
	return b.String()
}

// detailsSetImgFetchPriorityLow mirrors block_core_details_set_img_fetchpriority_low.
func detailsSetImgFetchPriorityLow(htmlFragment string) string {
	return reImgTagOpen.ReplaceAllStringFunc(htmlFragment, func(tag string) string {
		lower := strings.ToLower(tag)
		if strings.Contains(lower, "fetchpriority=") {
			re := regexp.MustCompile(`(?i)\s*fetchpriority\s*=\s*(["']).*?\1`)
			tag = re.ReplaceAllString(tag, "")
		}
		// Insert before trailing > or />.
		if strings.HasSuffix(tag, "/>") {
			return strings.TrimSuffix(tag, "/>") + ` fetchpriority="low"/>`
		}
		if strings.HasSuffix(tag, ">") {
			return strings.TrimSuffix(tag, ">") + ` fetchpriority="low">`
		}
		return tag
	})
}

// ensureRootAttribute sets or replaces attr="value" on the first opening tag.
func ensureRootAttribute(htmlFragment, attr, value string) string {
	if htmlFragment == "" || attr == "" {
		return htmlFragment
	}
	i := strings.Index(htmlFragment, "<")
	if i < 0 {
		return htmlFragment
	}
	rest := htmlFragment[i+1:]
	if rest == "" || rest[0] == '/' || rest[0] == '!' {
		return htmlFragment
	}
	end := strings.Index(rest, ">")
	if end < 0 {
		return htmlFragment
	}
	open := rest[:end]
	selfClose := strings.HasSuffix(open, "/")
	if selfClose {
		open = strings.TrimSpace(strings.TrimSuffix(open, "/"))
	}
	re := regexp.MustCompile(`(?i)\s` + regexp.QuoteMeta(attr) + `\s*=\s*(["']).*?\1`)
	open = re.ReplaceAllString(open, "")
	open = strings.TrimRight(open, " \t")
	open += fmt.Sprintf(` %s="%s"`, attr, html.EscapeString(value))
	if selfClose {
		open += " /"
	}
	return htmlFragment[:i+1] + open + htmlFragment[i+1+end:]
}

// ensureRootBooleanAttribute ensures a boolean attribute (e.g. open) on the root tag.
func ensureRootBooleanAttribute(htmlFragment, attr string) string {
	if htmlFragment == "" || attr == "" {
		return htmlFragment
	}
	i := strings.Index(htmlFragment, "<")
	if i < 0 {
		return htmlFragment
	}
	rest := htmlFragment[i+1:]
	if rest == "" || rest[0] == '/' || rest[0] == '!' {
		return htmlFragment
	}
	end := strings.Index(rest, ">")
	if end < 0 {
		return htmlFragment
	}
	open := rest[:end]
	selfClose := strings.HasSuffix(open, "/")
	if selfClose {
		open = strings.TrimSpace(strings.TrimSuffix(open, "/"))
	}
	lower := strings.ToLower(open)
	token := " " + strings.ToLower(attr)
	if strings.Contains(lower, token+"=") || strings.Contains(lower, token+" ") || strings.HasSuffix(lower, token) {
		return htmlFragment
	}
	open = strings.TrimRight(open, " \t") + " " + attr
	if selfClose {
		open += " /"
	}
	return htmlFragment[:i+1] + open + htmlFragment[i+1+end:]
}
