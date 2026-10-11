package blocks

import (
	"html"
	"strconv"
	"strings"
)

// CategoryItem is a taxonomy term for core/categories (P5.81).
type CategoryItem struct {
	Name   string
	URL    string
	Count  int
	Parent int64
	ID     int64
}

// renderCategories renders core/categories as a list (P5.81).
// Dropdown mode is deferred; list markup matches render_block_core_categories.
func (e *Env) renderCategories(block Block) string {
	e = e.ensure()
	// P5.167: coerce supports.align before class assembly.
	block.Attrs = sanitizeCategoriesBlockAttrs(block.Attrs)
	taxonomy := strings.TrimSpace(attrString(block.Attrs, "taxonomy"))
	if taxonomy == "" {
		taxonomy = "category"
	}
	showCounts := false
	if v, ok := block.Attrs["showPostCounts"].(bool); ok {
		showCounts = v
	}
	showEmpty := false
	if v, ok := block.Attrs["showEmpty"].(bool); ok {
		showEmpty = v
	}
	topLevelOnly := false
	if v, ok := block.Attrs["showOnlyTopLevel"].(bool); ok {
		topLevelOnly = v
	}
	showHierarchy := false
	if v, ok := block.Attrs["showHierarchy"].(bool); ok {
		showHierarchy = v
	}

	var items []CategoryItem
	if e.ListCategories != nil {
		items = e.ListCategories(taxonomy)
	}
	filtered := make([]CategoryItem, 0, len(items))
	for _, it := range items {
		if !showEmpty && it.Count <= 0 {
			continue
		}
		if topLevelOnly && it.Parent != 0 {
			continue
		}
		filtered = append(filtered, it)
	}
	if len(filtered) == 0 {
		return ""
	}

	classes := []string{
		"wp-block-categories-list",
		"wp-block-categories",
		"wp-block-categories-taxonomy-" + taxonomy,
	}
	// P5.167: supports.align true → left|center|right|wide|full.
	if align := strings.TrimSpace(attrString(block.Attrs, "align")); align != "" {
		classes = append(classes, "align"+sanitizeClassToken(align))
	}
	var b strings.Builder
	b.WriteString(`<ul class="`)
	b.WriteString(html.EscapeString(strings.Join(classes, " ")))
	b.WriteString(`">`)
	if showHierarchy && !topLevelOnly {
		writeCategoryTree(&b, filtered, 0, showCounts)
	} else {
		for _, it := range filtered {
			writeCategoryLI(&b, it, showCounts)
		}
	}
	b.WriteString(`</ul>`)
	return b.String()
}

// sanitizeCategoriesBlockAttrs coerces core/categories supports.align (P5.167).
// Allowed: left, center, right, wide, full. Other values (incl. none) are dropped.
func sanitizeCategoriesBlockAttrs(attrs map[string]any) map[string]any {
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
	default:
		delete(out, "align")
	}
	return out
}

func writeCategoryTree(b *strings.Builder, items []CategoryItem, parent int64, showCounts bool) {
	for _, it := range items {
		if it.Parent != parent {
			continue
		}
		writeCategoryOpenLI(b, it, showCounts)
		hasKids := false
		for _, c := range items {
			if c.Parent == it.ID {
				hasKids = true
				break
			}
		}
		if hasKids {
			b.WriteString(`<ul class="children">`)
			writeCategoryTree(b, items, it.ID, showCounts)
			b.WriteString(`</ul>`)
		}
		b.WriteString(`</li>`)
	}
}

func writeCategoryLI(b *strings.Builder, it CategoryItem, showCounts bool) {
	writeCategoryOpenLI(b, it, showCounts)
	b.WriteString(`</li>`)
}

func writeCategoryOpenLI(b *strings.Builder, it CategoryItem, showCounts bool) {
	b.WriteString(`<li class="cat-item cat-item-`)
	b.WriteString(strconv.FormatInt(it.ID, 10))
	b.WriteString(`"><a href="`)
	b.WriteString(html.EscapeString(it.URL))
	b.WriteString(`">`)
	b.WriteString(html.EscapeString(it.Name))
	b.WriteString(`</a>`)
	if showCounts {
		b.WriteString(` (`)
		b.WriteString(strconv.Itoa(it.Count))
		b.WriteString(`)`)
	}
}
