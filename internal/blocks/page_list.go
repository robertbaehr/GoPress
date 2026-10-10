package blocks

import (
	"html"
	"strings"
)

// renderPageList renders core/page-list (P5.3 / P5.75).
// Honors parentPageID, nests children, and emits WP page-list item classes.
// When used inside navigation, items also carry navigation-item classes.
func (e *Env) renderPageList(block Block) string {
	e = e.ensure()
	if e.ListPages == nil {
		return ""
	}
	pages := e.ListPages()
	if len(pages) == 0 {
		return ""
	}
	parentID := attrInt64(block.Attrs, "parentPageID")
	return renderPageListItems(pages, parentID, e.CurrentPageID, e.FrontPageID, e.renderingNavigation)
}

// renderPageListItems builds nested <li> markup for page-list.
func renderPageListItems(pages []NavLink, parentID, currentID, frontPageID int64, inNav bool) string {
	byParent := map[int64][]NavLink{}
	for _, p := range pages {
		if strings.TrimSpace(p.Label) == "" {
			continue
		}
		byParent[p.Parent] = append(byParent[p.Parent], p)
	}
	roots := byParent[parentID]
	if parentID != 0 && len(roots) == 0 {
		return ""
	}
	return renderPageListLevel(roots, byParent, currentID, frontPageID, inNav)
}

func renderPageListLevel(level []NavLink, byParent map[int64][]NavLink, currentID, frontPageID int64, inNav bool) string {
	var b strings.Builder
	for _, page := range level {
		// ID==0 means flat fallback links (tests/legacy); never nest under self.
		var children []NavLink
		if page.ID > 0 {
			children = byParent[page.ID]
		}
		classes := []string{"wp-block-pages-list__item"}
		if inNav {
			classes = append(classes, "wp-block-navigation-item")
		}
		if page.ID > 0 && page.ID == currentID {
			classes = append(classes, "current-menu-item")
		}
		if len(children) > 0 {
			classes = append(classes, "has-child")
		}
		if page.ID > 0 && frontPageID > 0 && page.ID == frontPageID {
			classes = append(classes, "menu-item-home")
		}
		b.WriteString(`<li class="`)
		b.WriteString(html.EscapeString(strings.Join(classes, " ")))
		b.WriteString(`">`)
		linkClass := "wp-block-pages-list__item__link"
		if inNav {
			linkClass += " wp-block-navigation-item__content"
		}
		b.WriteString(`<a class="`)
		b.WriteString(html.EscapeString(linkClass))
		b.WriteString(`"`)
		if page.URL != "" {
			b.WriteString(` href="`)
			b.WriteString(html.EscapeString(page.URL))
			b.WriteString(`"`)
		}
		if page.ID > 0 && page.ID == currentID {
			b.WriteString(` aria-current="page"`)
		}
		b.WriteString(`>`)
		b.WriteString(html.EscapeString(page.Label))
		b.WriteString(`</a>`)
		if len(children) > 0 {
			subClass := "wp-block-page-list"
			if inNav {
				subClass = "wp-block-navigation__submenu-container"
			}
			b.WriteString(`<ul class="`)
			b.WriteString(html.EscapeString(subClass))
			b.WriteString(`">`)
			b.WriteString(renderPageListLevel(children, byParent, currentID, frontPageID, inNav))
			b.WriteString(`</ul>`)
		}
		b.WriteString(`</li>`)
	}
	return b.String()
}
