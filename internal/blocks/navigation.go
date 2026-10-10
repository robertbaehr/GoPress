package blocks

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

func (e *Env) renderNavigation(block Block) string {
	e = e.ensure()
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
		inner = renderNavLinks(e.ListPages())
	}
	if strings.TrimSpace(inner) == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString(`<nav class="wp-block-navigation">`)
	b.WriteString(`<ul class="wp-block-navigation__container">`)
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

func (e *Env) renderNavigationLink(block Block) string {
	label, _ := block.Attrs["label"].(string)
	url, _ := block.Attrs["url"].(string)
	if strings.TrimSpace(label) == "" {
		return ""
	}
	return navItemHTML(label, url)
}

func (e *Env) renderPageList(_ Block) string {
	if e.ListPages == nil {
		return ""
	}
	return renderNavLinks(e.ListPages())
}

func renderNavLinks(links []NavLink) string {
	var b strings.Builder
	for _, link := range links {
		if strings.TrimSpace(link.Label) == "" {
			continue
		}
		b.WriteString(navItemHTML(link.Label, link.URL))
	}
	return b.String()
}

func navItemHTML(label, url string) string {
	var b strings.Builder
	b.WriteString(`<li class="wp-block-navigation-item"><a class="wp-block-navigation-item__content"`)
	if url != "" {
		b.WriteString(` href="`)
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
