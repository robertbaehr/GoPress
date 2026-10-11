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

// renderList renders core/list (P5.63 / P5.104 / P5.147).
// Prefers saved markup; ensures wp-block-list on ul/ol (block_core_list_render);
// synthesizes from ordered/values/start/reversed/type/className/anchor when needed.
// P5.147: coerce align to wide|full; gradient preset + fontSize polish; ordered attrs on saved ol.
// Color/typography/spacing land via applySupports on the caller (sanitized attrs).
func (e *Env) renderList(block Block) string {
	e = e.ensure()
	attrs := sanitizeListBlockAttrs(block.Attrs)
	block.Attrs = attrs
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && reListTag.MatchString(out) {
		return polishListMarkup(out, attrs)
	}
	inner := ""
	if e != nil {
		inner = e.renderInnerBlocks(block)
	}
	if strings.TrimSpace(inner) == "" {
		inner = strings.TrimSpace(attrString(attrs, "values"))
	}
	return synthesizeListBlock(attrs, inner)
}

// sanitizeListBlockAttrs coerces core/list supports.align to the block.json enum (P5.147).
// Allowed: wide, full. Other align values are dropped before synthesize/polish/supports.
func sanitizeListBlockAttrs(attrs map[string]any) map[string]any {
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

// polishListMarkup ensures wp-block-list + align/className/anchor/gradient/fontSize
// on saved HTML (P5.104 / P5.147). Ordered lists also get start/reversed/type.
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
	// P5.147: typography.fontSize preset class (also via applySupports).
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		out = addRootClass(out, "has-"+sanitizeClassToken(fs)+"-font-size")
	}
	// P5.147: color.gradients preset → has-{slug}-gradient-background.
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		out = addRootClass(out, "has-background")
		out = addRootClass(out, "has-"+sanitizeClassToken(g)+"-gradient-background")
	}
	// P5.147: ordered list attrs on saved <ol> markup.
	if attrBool(attrs, "ordered") || reListOrderedTag.MatchString(out) {
		if start, ok := attrInt(attrs, "start"); ok {
			out = ensureRootAttribute(out, "start", strconv.Itoa(start))
		}
		if attrBool(attrs, "reversed") {
			out = ensureRootBooleanAttribute(out, "reversed")
		}
		if typ := strings.TrimSpace(attrString(attrs, "type")); typ != "" {
			out = ensureRootAttribute(out, "type", typ)
		}
	}
	return out
}

var reListOrderedTag = regexp.MustCompile(`(?i)<ol\b`)

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
	if fs := strings.TrimSpace(attrString(attrs, "fontSize")); fs != "" {
		class += " has-" + sanitizeClassToken(fs) + "-font-size"
	}
	if g := strings.TrimSpace(attrString(attrs, "gradient")); g != "" {
		class += " has-background has-" + sanitizeClassToken(g) + "-gradient-background"
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
