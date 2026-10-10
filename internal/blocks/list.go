package blocks

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reListTag     = regexp.MustCompile(`(?i)<(ul|ol)\b`)
	reListItemTag = regexp.MustCompile(`(?i)<li\b`)
)

// renderList renders core/list (P5.63 / P5.104).
// Prefers saved markup; ensures wp-block-list on ul/ol (block_core_list_render);
// synthesizes from ordered/values/start/reversed/type/className/anchor when needed.
// Color/typography/spacing land via applySupports on the caller.
func (e *Env) renderList(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reListTag.MatchString(out) {
		return polishListMarkup(out, block.Attrs)
	}
	inner := ""
	if e != nil {
		inner = e.renderInnerBlocks(block)
	}
	if strings.TrimSpace(inner) == "" {
		inner = strings.TrimSpace(attrString(block.Attrs, "values"))
	}
	return synthesizeListBlock(block.Attrs, inner)
}

// polishListMarkup ensures wp-block-list + align/className/anchor on saved HTML (P5.104).
func polishListMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-list")
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		out = addRootClass(out, "align"+sanitizeClassToken(align))
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

func synthesizeListBlock(attrs map[string]any, inner string) string {
	ordered := attrBool(attrs, "ordered")
	tag := "ul"
	if ordered {
		tag = "ol"
	}
	class := listClassFromAttrs(attrs)
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	b.WriteString(` class="`)
	b.WriteString(class)
	b.WriteString(`"`)
	if ordered {
		if start, ok := attrInt(attrs, "start"); ok {
			b.WriteString(` start="`)
			b.WriteString(strconv.Itoa(start))
			b.WriteString(`"`)
		}
		if attrBool(attrs, "reversed") {
			b.WriteString(" reversed")
		}
		if typ := strings.TrimSpace(attrString(attrs, "type")); typ != "" {
			b.WriteString(` type="`)
			b.WriteString(html.EscapeString(typ))
			b.WriteString(`"`)
		}
	}
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(">")
	b.WriteString(inner)
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">")
	return b.String()
}

func listClassFromAttrs(attrs map[string]any) string {
	class := "wp-block-list"
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		class += " align" + sanitizeClassToken(align)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		class += " " + html.EscapeString(cn)
	}
	return class
}

// renderListItem renders core/list-item (P5.63 / P5.104).
// Prefers saved markup; synthesizes <li> from content; honors anchor.
// Color/typography/spacing land via applySupports on the caller.
func (e *Env) renderListItem(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reListItemTag.MatchString(out) {
		return polishListItemMarkup(out, block.Attrs)
	}
	inner := ""
	if e != nil {
		inner = e.renderInnerBlocks(block)
	}
	if strings.TrimSpace(inner) == "" {
		inner = attrString(block.Attrs, "content")
	}
	return synthesizeListItemBlock(block.Attrs, inner)
}

// polishListItemMarkup ensures anchor on saved <li> markup (P5.104).
func polishListItemMarkup(out string, attrs map[string]any) string {
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		out = ensureRootAttribute(out, "id", anchor)
	}
	return out
}

func synthesizeListItemBlock(attrs map[string]any, inner string) string {
	var b strings.Builder
	b.WriteString("<li")
	if anchor := strings.TrimSpace(attrString(attrs, "anchor")); anchor != "" {
		b.WriteString(` id="`)
		b.WriteString(html.EscapeString(anchor))
		b.WriteString(`"`)
	}
	b.WriteString(">")
	b.WriteString(inner)
	b.WriteString("</li>")
	return b.String()
}
