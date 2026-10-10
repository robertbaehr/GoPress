package plugins

import (
	"html"
	"net/url"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// ActionLinks builds the default activate/deactivate row actions and applies
// plugin_action_links / plugin_action_links_{$plugin} filters (P9.14).
// Returned map keys are action slugs; values are HTML for the link (or span).
func ActionLinks(c *wp.Context, p PluginData, context string) map[string]string {
	actions := map[string]string{}
	if c == nil {
		return actions
	}
	active := IsPluginActive(c, p.File)
	if active {
		nonce := c.CreateNonce("deactivate-plugin_" + p.File)
		href := "/wp-admin/plugins.php?action=deactivate&plugin=" + url.QueryEscape(p.File) + "&_wpnonce=" + url.QueryEscape(nonce)
		actions["deactivate"] = `<a href="` + html.EscapeString(href) + `">Deactivate</a>`
	} else {
		nonce := c.CreateNonce("activate-plugin_" + p.File)
		href := "/wp-admin/plugins.php?action=activate&plugin=" + url.QueryEscape(p.File) + "&_wpnonce=" + url.QueryEscape(nonce)
		actions["activate"] = `<a href="` + html.EscapeString(href) + `">Activate</a>`
		// P9.18: Uninstall button for inactive uninstallable plugins (hooks stub;
		// delete_plugins / filesystem delete remains out of scope).
		if IsUninstallablePlugin(c, p.File) {
			unNonce := c.CreateNonce("uninstall-plugin_" + p.File)
			unHref := "/wp-admin/plugins.php?action=uninstall&plugin=" + url.QueryEscape(p.File) + "&_wpnonce=" + url.QueryEscape(unNonce)
			actions["uninstall"] = `<a href="` + html.EscapeString(unHref) + `" class="delete" aria-label="Uninstall ` + html.EscapeString(p.Name) + `">Uninstall</a>`
		}
	}
	if context == "" {
		context = "all"
	}
	pluginData := map[string]string{
		"Name":        p.Name,
		"Description": p.Description,
		"Author":      p.Author,
		"AuthorURI":   p.AuthorURI,
		"PluginURI":   p.PluginURI,
		"Version":     p.Version,
		"TextDomain":  p.TextDomain,
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("plugin_action_links", actions, p.File, pluginData, context); v != nil {
			if m, ok := v.(map[string]string); ok {
				actions = m
			}
		}
		tag := "plugin_action_links_" + p.File
		if v := c.Hooks.ApplyFilters(tag, actions, p.File, pluginData, context); v != nil {
			if m, ok := v.(map[string]string); ok {
				actions = m
			}
		}
	}
	return actions
}

// FormatActionLinksHTML joins action link map into WP row-actions spans.
// Order prefers activate, deactivate, then remaining keys sorted by insertion-ish name.
func FormatActionLinksHTML(actions map[string]string) string {
	if len(actions) == 0 {
		return ""
	}
	order := []string{"activate", "deactivate", "edit", "delete", "uninstall"}
	seen := map[string]bool{}
	var parts []string
	for _, key := range order {
		htmlPart, ok := actions[key]
		if !ok || strings.TrimSpace(htmlPart) == "" {
			continue
		}
		parts = append(parts, `<span class="`+html.EscapeString(key)+`">`+htmlPart+`</span>`)
		seen[key] = true
	}
	for key, htmlPart := range actions {
		if seen[key] || strings.TrimSpace(htmlPart) == "" {
			continue
		}
		parts = append(parts, `<span class="`+html.EscapeString(key)+`">`+htmlPart+`</span>`)
	}
	return strings.Join(parts, " | ")
}
