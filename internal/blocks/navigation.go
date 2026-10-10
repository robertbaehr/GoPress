package blocks

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

func (e *Env) renderNavigation(block Block) string {
	e = e.ensure()
	prevNav := e.renderingNavigation
	e.renderingNavigation = true
	defer func() { e.renderingNavigation = prevNav }()
	inner := ""

	ref := attrInt64(block.Attrs, "ref")
	if ref == 0 {
		ref = attrInt64(block.Attrs, "navigationMenuId") // legacy
	}

	switch {
	case ref > 0 && e.LoadNavigation != nil:
		if e.seenNav == nil {
			e.seenNav = map[int64]bool{}
		}
		if e.seenNav[ref] {
			return ""
		}
		e.seenNav[ref] = true
		content, ok := e.LoadNavigation(ref)
		delete(e.seenNav, ref)
		if ok && strings.TrimSpace(content) != "" {
			inner = e.renderNavigationInnerMarkup(content)
		}
	case len(block.InnerBlocks) > 0:
		inner = e.renderNavigationItems(block.InnerBlocks)
	}

	if strings.TrimSpace(inner) == "" && e.ListPages != nil {
		inner = e.renderPageList(Block{})
	}
	if strings.TrimSpace(inner) == "" {
		return ""
	}

	ariaLabel, _ := block.Attrs["ariaLabel"].(string)
	layoutOrientation := ""
	if layout, ok := block.Attrs["layout"].(map[string]any); ok {
		if o, ok := layout["orientation"].(string); ok {
			layoutOrientation = strings.TrimSpace(o)
		}
	}
	navClass := "wp-block-navigation"
	ulClass := "wp-block-navigation__container"
	if layoutOrientation == "vertical" {
		navClass = "is-vertical " + navClass
		ulClass += " is-vertical wp-block-navigation"
	}
	var b strings.Builder
	b.WriteString(`<nav class="`)
	b.WriteString(html.EscapeString(navClass))
	b.WriteString(`"`)
	if ariaLabel != "" {
		b.WriteString(` aria-label="`)
		b.WriteString(html.EscapeString(ariaLabel))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(`<ul class="`)
	b.WriteString(html.EscapeString(ulClass))
	b.WriteString(`"`)
	if ariaLabel != "" {
		b.WriteString(` aria-label="`)
		b.WriteString(html.EscapeString(ariaLabel))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	b.WriteString(inner)
	b.WriteString(`</ul></nav>`)
	return b.String()
}

// renderNavigationInnerMarkup renders wp_navigation post content into <li> items.
func (e *Env) renderNavigationInnerMarkup(content string) string {
	return e.renderNavigationItems(Parse(content))
}

func (e *Env) renderNavigationItems(items []Block) string {
	var b strings.Builder
	for _, item := range items {
		switch blockName(item) {
		case "core/navigation-link":
			b.WriteString(e.renderNavigationLink(item))
		case "core/home-link":
			b.WriteString(e.renderHomeLink(item))
		case "core/loginout":
			b.WriteString(e.renderLoginout(item))
		case "core/page-list":
			b.WriteString(e.renderPageList(item))
		case "core/navigation-submenu":
			// Basic: render as a link plus nested list if present.
			b.WriteString(e.renderNavigationLink(item))
		case "":
			// ignore freeform
		default:
			// Nested navigation or other blocks: try saved / recursive.
			if len(item.InnerBlocks) > 0 {
				b.WriteString(e.renderNavigationItems(item.InnerBlocks))
			}
		}
	}
	return b.String()
}

// renderHomeLink renders core/home-link (P5.73) — a nav item that always
// points at home_url(), with current-menu-item when on the front page.
func (e *Env) renderHomeLink(block Block) string {
	e = e.ensure()
	label, _ := block.Attrs["label"].(string)
	label = strings.TrimSpace(label)
	if label == "" {
		label = "Home"
	}
	home := ""
	if e.HomeURL != nil {
		home = strings.TrimRight(strings.TrimSpace(e.HomeURL()), "/")
		if home == "" {
			home = "/"
		}
	} else {
		home = "/"
	}
	isCurrent := e.IsFrontPage != nil && e.IsFrontPage()

	liClass := "wp-block-home-link wp-block-navigation-item"
	if isCurrent {
		liClass += " current-menu-item"
	}

	var b strings.Builder
	b.WriteString(`<li class="`)
	b.WriteString(html.EscapeString(liClass))
	b.WriteString(`"><a class="wp-block-home-link__content wp-block-navigation-item__content" href="`)
	b.WriteString(html.EscapeString(home))
	b.WriteString(`" rel="home"`)
	if opensInNewTab(block.Attrs) {
		b.WriteString(` target="_blank"`)
	}
	if isCurrent {
		b.WriteString(` aria-current="page"`)
	}
	b.WriteString(`>`)
	b.WriteString(html.EscapeString(label))
	if desc, _ := block.Attrs["description"].(string); strings.TrimSpace(desc) != "" {
		b.WriteString(`<span class="wp-block-navigation-item__description">`)
		b.WriteString(html.EscapeString(desc))
		b.WriteString(`</span>`)
	}
	b.WriteString(`</a></li>`)
	return b.String()
}

func opensInNewTab(attrs map[string]any) bool {
	if attrs == nil {
		return false
	}
	v, ok := attrs["opensInNewTab"]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}

func (e *Env) renderNavigationLink(block Block) string {
	label, _ := block.Attrs["label"].(string)
	url, _ := block.Attrs["url"].(string)
	if strings.TrimSpace(label) == "" {
		return ""
	}
	return navItemHTML(label, url)
}

func navItemHTML(label, url string) string {
	var b strings.Builder
	b.WriteString(`<li class="wp-block-navigation-item wp-block-navigation-link"><a class="wp-block-navigation-item__content"`)
	if url != "" {
		b.WriteString(`  href="`)
		b.WriteString(html.EscapeString(url))
		b.WriteString(`"`)
	}
	b.WriteString(`><span class="wp-block-navigation-item__label">`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</span></a></li>`)
	return b.String()
}

func attrInt64(attrs map[string]any, key string) int64 {
	if attrs == nil {
		return 0
	}
	v, ok := attrs[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case jsonNumber:
		n, _ := t.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		n, err := strconv.ParseInt(fmt.Sprint(t), 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
}

// jsonNumber matches encoding/json.Number without importing for every call site.
type jsonNumber interface {
	Int64() (int64, error)
}
