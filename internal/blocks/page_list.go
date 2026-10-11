package blocks

import (
	"html"
	"strings"
)

// pageListNavCtx carries Navigation block context for page-list children (P5.94).
type pageListNavCtx struct {
	showSubmenuIcon   bool
	submenuVisibility string // hover | click | always
	textColor         string
	backgroundColor   string
	customTextColor   string
	customBackground  string
}

// renderPageList renders core/page-list (P5.3 / P5.75 / P5.94).
// Honors parentPageID, nests children, emits WP page-list item classes,
// current-menu-ancestor, (no title), wrapper className/has-link-color,
// and nav submenu open/icon classes from Navigation context.
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
	return renderPageListItems(pages, parentID, e.CurrentPageID, e.FrontPageID, e.renderingNavigation, e.pageListNav)
}

// renderPageListItems builds nested <li> markup for page-list.
func renderPageListItems(pages []NavLink, parentID, currentID, frontPageID int64, inNav bool, nav pageListNavCtx) string {
	byParent := map[int64][]NavLink{}
	byID := map[int64]NavLink{}
	for _, p := range pages {
		byParent[p.Parent] = append(byParent[p.Parent], p)
		if p.ID > 0 {
			byID[p.ID] = p
		}
	}
	ancestorIDs := pageListAncestorIDs(byID, currentID)
	roots := byParent[parentID]
	if parentID != 0 && len(roots) == 0 {
		return ""
	}
	return renderPageListLevel(roots, byParent, currentID, frontPageID, ancestorIDs, inNav, nav)
}

// pageListAncestorIDs walks Parent links for the current page (P5.94).
func pageListAncestorIDs(byID map[int64]NavLink, currentID int64) map[int64]bool {
	out := map[int64]bool{}
	if currentID <= 0 {
		return out
	}
	seen := map[int64]bool{}
	id := currentID
	for {
		p, ok := byID[id]
		if !ok || p.Parent <= 0 || seen[p.Parent] {
			break
		}
		seen[p.Parent] = true
		out[p.Parent] = true
		id = p.Parent
	}
	return out
}

func renderPageListLevel(level []NavLink, byParent map[int64][]NavLink, currentID, frontPageID int64, ancestors map[int64]bool, inNav bool, nav pageListNavCtx) string {
	var b strings.Builder
	openOnClick := inNav && nav.submenuVisibility == "click"
	openOnHover := inNav && nav.submenuVisibility == "hover"
	openAlways := inNav && nav.submenuVisibility == "always"
	for _, page := range level {
		// ID==0 means flat fallback links (tests/legacy); never nest under self.
		var children []NavLink
		if page.ID > 0 {
			children = byParent[page.ID]
		}
		classes := []string{"wp-block-pages-list__item"}
		if inNav {
			classes = append(classes, "wp-block-navigation-item")
			if len(children) > 0 {
				if openOnClick {
					classes = append(classes, "open-on-click")
				} else if openOnHover && nav.showSubmenuIcon {
					classes = append(classes, "open-on-hover-click")
				} else if openAlways {
					classes = append(classes, "open-always")
				}
			}
		}
		if page.ID > 0 && page.ID == currentID {
			classes = append(classes, "current-menu-item")
		}
		if page.ID > 0 && ancestors[page.ID] {
			classes = append(classes, "current-menu-ancestor")
		}
		if len(children) > 0 {
			classes = append(classes, "has-child")
		}
		if page.ID > 0 && frontPageID > 0 && page.ID == frontPageID {
			classes = append(classes, "menu-item-home")
		}
		title := strings.TrimSpace(page.Label)
		if title == "" {
			title = "(no title)"
		}
		b.WriteString(`<li class="`)
		b.WriteString(html.EscapeString(strings.Join(classes, " ")))
		b.WriteString(`">`)
		linkClass := "wp-block-pages-list__item__link"
		if inNav {
			linkClass += " wp-block-navigation-item__content"
		}
		if openOnClick && len(children) > 0 {
			ariaLabel := title + " submenu"
			b.WriteString(`<button aria-label="`)
			b.WriteString(html.EscapeString(ariaLabel))
			b.WriteString(`" class="`)
			b.WriteString(html.EscapeString(linkClass + " wp-block-navigation-submenu__toggle"))
			b.WriteString(`" aria-expanded="false">`)
			b.WriteString(html.EscapeString(title))
			b.WriteString(`</button>`)
			b.WriteString(`<span class="wp-block-page-list__submenu-icon wp-block-navigation__submenu-icon">`)
			b.WriteString(pageListSubmenuSVG())
			b.WriteString(`</span>`)
		} else {
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
			b.WriteString(html.EscapeString(title))
			b.WriteString(`</a>`)
			if len(children) > 0 && inNav && nav.showSubmenuIcon && !openOnClick {
				ariaLabel := title + " submenu"
				b.WriteString(`<button aria-label="`)
				b.WriteString(html.EscapeString(ariaLabel))
				b.WriteString(`" class="wp-block-navigation__submenu-icon wp-block-navigation-submenu__toggle" aria-expanded="false">`)
				b.WriteString(pageListSubmenuSVG())
				b.WriteString(`</button>`)
			}
		}
		if len(children) > 0 {
			subClass := "wp-block-page-list"
			if inNav {
				subClass = "wp-block-navigation__submenu-container"
			}
			b.WriteString(`<ul class="`)
			b.WriteString(html.EscapeString(subClass))
			b.WriteString(`">`)
			b.WriteString(renderPageListLevel(children, byParent, currentID, frontPageID, ancestors, inNav, nav))
			b.WriteString(`</ul>`)
		}
		b.WriteString(`</li>`)
	}
	return b.String()
}

func pageListSubmenuSVG() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true" focusable="false"><path d="M1.50002 4L6.00002 8L10.5 4" stroke-width="1.5"></path></svg>`
}

// pageListWrapperAttrs builds class/style for the standalone <ul> wrapper (P5.94).
func pageListWrapperAttrs(attrs map[string]any) (classAttr, styleAttr string) {
	classes := []string{"wp-block-page-list"}
	style := ""
	if tc := strings.TrimSpace(attrString(attrs, "textColor")); tc != "" {
		classes = append(classes, "has-text-color", "has-"+sanitizeClassToken(tc)+"-color")
	} else if custom := styleColorText(attrs); custom != "" {
		classes = append(classes, "has-text-color")
		style += "color: " + custom + ";"
	}
	if bg := strings.TrimSpace(attrString(attrs, "backgroundColor")); bg != "" {
		classes = append(classes, "has-background", "has-"+sanitizeClassToken(bg)+"-background-color")
	} else if custom := styleColorBackground(attrs); custom != "" {
		classes = append(classes, "has-background")
		style += "background-color: " + custom + ";"
	}
	if hasLinkColorAttr(attrs) {
		classes = append(classes, "has-link-color")
	}
	if cn := strings.TrimSpace(attrString(attrs, "className")); cn != "" {
		classes = append(classes, cn)
	}
	classAttr = strings.Join(classes, " ")
	styleAttr = style
	return classAttr, styleAttr
}

// pageListNavCtxFromAttrs reads Navigation attrs into page-list nav context (P5.94 / P5.117).
// showSubmenuIcon defaults true (block.json); openSubmenusOnClick / submenuVisibility map to hover|click|always.
func pageListNavCtxFromAttrs(attrs map[string]any) pageListNavCtx {
	ctx := pageListNavCtx{submenuVisibility: "hover", showSubmenuIcon: true}
	if attrs == nil {
		return ctx
	}
	if v, ok := attrs["showSubmenuIcon"].(bool); ok {
		ctx.showSubmenuIcon = v
	}
	// Deprecated openSubmenusOnClick wins when present (block_core_page_list_get_submenu_visibility).
	if v, ok := attrs["openSubmenusOnClick"].(bool); ok {
		if v {
			ctx.submenuVisibility = "click"
		} else {
			ctx.submenuVisibility = "hover"
		}
	} else if sv := strings.TrimSpace(attrString(attrs, "submenuVisibility")); sv != "" {
		switch strings.ToLower(sv) {
		case "click", "hover", "always":
			ctx.submenuVisibility = strings.ToLower(sv)
		}
	}
	ctx.textColor = strings.TrimSpace(attrString(attrs, "textColor"))
	ctx.backgroundColor = strings.TrimSpace(attrString(attrs, "backgroundColor"))
	ctx.customTextColor = styleColorText(attrs)
	ctx.customBackground = styleColorBackground(attrs)
	return ctx
}
