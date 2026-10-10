package blocks

import (
	"html"
	"regexp"
	"strings"
)

var (
	reFigureTable = regexp.MustCompile(`(?is)<figure\b`)
	reTableTag    = regexp.MustCompile(`(?is)<table\b`)
)

// renderTable renders core/table (P5.66).
// Prefers saved markup; ensures figure.wp-block-table (+ align/className/style);
// ensures has-fixed-layout on <table> when hasFixedLayout is true (block default);
// synthesizes from head/body/foot/caption attrs when markup is absent.
func (e *Env) renderTable(block Block) string {
	e = e.ensure()
	out := e.renderSaved(block)
	if strings.TrimSpace(out) != "" && (reFigureTable.MatchString(out) || reTableTag.MatchString(out)) {
		return polishTableMarkup(out, block.Attrs)
	}
	return synthesizeTableBlock(block.Attrs)
}

func polishTableMarkup(out string, attrs map[string]any) string {
	out = addRootClass(out, "wp-block-table")
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		out = addRootClass(out, "align"+sanitizeClassToken(align))
	}
	if style := tableStyleClass(attrs); style != "" {
		out = addRootClass(out, style)
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		for _, part := range strings.Fields(cn) {
			out = addRootClass(out, part)
		}
	}
	if tableHasFixedLayout(attrs) {
		out = ensureTableFixedLayout(out)
	}
	return out
}

func synthesizeTableBlock(attrs map[string]any) string {
	head := tableSectionHTML(attrs, "head", "thead")
	body := tableSectionHTML(attrs, "body", "tbody")
	foot := tableSectionHTML(attrs, "foot", "tfoot")
	if head == "" && body == "" && foot == "" {
		return ""
	}

	figClass := "wp-block-table"
	if align := strings.TrimSpace(attrString(attrs, "align")); align != "" {
		figClass += " align" + sanitizeClassToken(align)
	}
	if style := tableStyleClass(attrs); style != "" {
		figClass += " " + style
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		figClass += " " + html.EscapeString(cn)
	}

	tableClass := ""
	if tableHasFixedLayout(attrs) {
		tableClass = ` class="has-fixed-layout"`
	}

	var b strings.Builder
	b.WriteString(`<figure class="`)
	b.WriteString(figClass)
	b.WriteString(`"><table`)
	b.WriteString(tableClass)
	b.WriteString(`>`)
	b.WriteString(head)
	b.WriteString(body)
	b.WriteString(foot)
	b.WriteString(`</table>`)
	if cap := strings.TrimSpace(attrString(attrs, "caption")); cap != "" {
		b.WriteString(`<figcaption class="wp-element-caption">`)
		b.WriteString(cap)
		b.WriteString(`</figcaption>`)
	}
	b.WriteString(`</figure>`)
	return b.String()
}

// tableHasFixedLayout mirrors block.json default true for hasFixedLayout.
func tableHasFixedLayout(attrs map[string]any) bool {
	if attrs == nil {
		return true
	}
	v, ok := attrs["hasFixedLayout"]
	if !ok || v == nil {
		return true
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		if s == "false" || s == "0" || s == "" {
			return false
		}
		return true
	case float64:
		return t != 0
	case int:
		return t != 0
	default:
		return true
	}
}

func tableStyleClass(attrs map[string]any) string {
	// Block style "stripes" → is-style-stripes (also often stored in className).
	style := strings.TrimSpace(attrString(attrs, "className"))
	if strings.Contains(style, "is-style-stripes") {
		return ""
	}
	// Some serialized forms store style via className only; honor explicit style attr if present.
	if s := strings.TrimSpace(attrString(attrs, "style")); s == "stripes" {
		return "is-style-stripes"
	}
	return ""
}

func tableSectionHTML(attrs map[string]any, key, tag string) string {
	raw, ok := attrs[key]
	if !ok || raw == nil {
		return ""
	}
	rows, ok := raw.([]any)
	if !ok || len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	b.WriteString(">")
	for _, rowAny := range rows {
		row, ok := rowAny.(map[string]any)
		if !ok {
			continue
		}
		cellsAny, _ := row["cells"].([]any)
		b.WriteString("<tr>")
		for _, cellAny := range cellsAny {
			cell, ok := cellAny.(map[string]any)
			if !ok {
				continue
			}
			cellTag := strings.TrimSpace(attrString(cell, "tag"))
			if cellTag == "" {
				if tag == "thead" {
					cellTag = "th"
				} else {
					cellTag = "td"
				}
			}
			b.WriteString("<")
			b.WriteString(cellTag)
			if scope := strings.TrimSpace(attrString(cell, "scope")); scope != "" {
				b.WriteString(` scope="`)
				b.WriteString(html.EscapeString(scope))
				b.WriteString(`"`)
			}
			if align := strings.TrimSpace(attrString(cell, "align")); align != "" {
				b.WriteString(` data-align="`)
				b.WriteString(html.EscapeString(align))
				b.WriteString(`"`)
			}
			if colspan := strings.TrimSpace(attrString(cell, "colspan")); colspan != "" {
				b.WriteString(` colspan="`)
				b.WriteString(html.EscapeString(colspan))
				b.WriteString(`"`)
			}
			if rowspan := strings.TrimSpace(attrString(cell, "rowspan")); rowspan != "" {
				b.WriteString(` rowspan="`)
				b.WriteString(html.EscapeString(rowspan))
				b.WriteString(`"`)
			}
			b.WriteString(">")
			b.WriteString(attrString(cell, "content"))
			b.WriteString("</")
			b.WriteString(cellTag)
			b.WriteString(">")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">")
	return b.String()
}

// ensureTableFixedLayout adds has-fixed-layout to the first <table> opening tag.
func ensureTableFixedLayout(htmlFragment string) string {
	loc := reTableTag.FindStringIndex(htmlFragment)
	if loc == nil {
		return htmlFragment
	}
	// Work on the opening tag only.
	rest := htmlFragment[loc[0]:]
	end := strings.Index(rest, ">")
	if end < 0 {
		return htmlFragment
	}
	open := rest[:end]
	if strings.Contains(strings.ToLower(open), "has-fixed-layout") {
		return htmlFragment
	}
	// Reuse addRootClass by isolating the table open… but class may already exist.
	patched := addRootClass(open+">", "has-fixed-layout")
	patched = strings.TrimSuffix(patched, ">")
	return htmlFragment[:loc[0]] + patched + htmlFragment[loc[0]+end:]
}
