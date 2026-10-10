package plugins

import (
	"html"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// RowMeta builds the Version | By Author | Visit plugin site meta line parts
// and applies the plugin_row_meta filter (P9.15).
// Returned slice elements are already HTML-safe (or filter-supplied HTML).
func RowMeta(c *wp.Context, p PluginData, status string) []string {
	meta := []string{}
	if p.Version != "" {
		meta = append(meta, "Version "+html.EscapeString(p.Version))
	}
	if p.Author != "" {
		author := html.EscapeString(p.Author)
		if p.AuthorURI != "" {
			author = `<a href="` + html.EscapeString(p.AuthorURI) + `">` + author + `</a>`
		}
		meta = append(meta, "By "+author)
	}
	if p.PluginURI != "" {
		meta = append(meta, `<a href="`+html.EscapeString(p.PluginURI)+`" target="_blank">Visit plugin site</a>`)
	}
	if status == "" {
		status = "all"
	}
	pluginData := map[string]string{
		"Name":        p.Name,
		"Description": p.Description,
		"Author":      p.Author,
		"AuthorURI":   p.AuthorURI,
		"PluginURI":   p.PluginURI,
		"Version":     p.Version,
		"TextDomain":  p.TextDomain,
		"Title":       p.Name,
		"AuthorName":  p.Author,
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("plugin_row_meta", meta, p.File, pluginData, status); v != nil {
			if m, ok := v.([]string); ok {
				meta = m
			}
		}
		c.Hooks.DoAction("after_plugin_row_meta", p.File, pluginData)
	}
	return meta
}

// FormatRowMetaHTML joins meta parts with " | " like WP_Plugins_List_Table.
func FormatRowMetaHTML(meta []string) string {
	parts := make([]string, 0, len(meta))
	for _, p := range meta {
		if strings.TrimSpace(p) == "" {
			continue
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, " | ")
}
