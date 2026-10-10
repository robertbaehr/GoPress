package blocks

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reListTag     = regexp.MustCompile(`(?i)<(ul|ol)\b`)
	reListItemTag = regexp.MustCompile(`(?i)<li\b`)
)

// renderList renders core/list (P5.63).
// Prefers saved markup; ensures wp-block-list on ul/ol (block_core_list_render);
// synthesizes from ordered/values/start/reversed/type/className when needed.
func (e *Env) renderList(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reListTag.MatchString(out) {
		return addRootClass(out, "wp-block-list")
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

// renderListItem renders core/list-item (P5.63).
// Prefers saved markup; synthesizes <li> from content attribute.
func (e *Env) renderListItem(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reListItemTag.MatchString(out) {
		return out
	}
	inner := ""
	if e != nil {
		inner = e.renderInnerBlocks(block)
	}
	if strings.TrimSpace(inner) == "" {
		inner = attrString(block.Attrs, "content")
	}
	return fmt.Sprintf("<li>%s</li>", inner)
}
