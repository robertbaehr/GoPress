// oEmbed discovery links (P3.20 / P3.92) — subset of wp_oembed_add_discovery_links.
package wp

import (
	"html"
	"net/url"
	"strings"
)

// GetOEmbedEndpointURL builds the oEmbed REST endpoint URL (get_oembed_endpoint_url).
// format is "json" (default) or "xml"; json omits the format query arg like WP.
func GetOEmbedEndpointURL(home, permalink, format string) string {
	return GetOEmbedEndpointURLWithHooks(nil, home, permalink, format)
}

// GetOEmbedEndpointURLWithHooks applies the oembed_endpoint_url filter (P3.92).
func GetOEmbedEndpointURLWithHooks(h *Hooks, home, permalink, format string) string {
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	permalink = strings.TrimSpace(permalink)
	format = strings.TrimSpace(format)
	if format == "" {
		format = "json"
	}
	endpoint := home + "/wp-json/oembed/1.0/embed"
	if home == "" {
		endpoint = "/wp-json/oembed/1.0/embed"
	}
	if permalink != "" {
		endpoint += "?url=" + url.QueryEscape(permalink)
		if format != "json" {
			endpoint += "&format=" + url.QueryEscape(format)
		}
	}
	if h != nil {
		filtered := h.ApplyFilters("oembed_endpoint_url", endpoint, permalink, format)
		if s, ok := filtered.(string); ok {
			endpoint = s
		}
	}
	return endpoint
}

// OEmbedDiscoveryLinks returns the JSON/XML oEmbed alternate <link> tags for a
// singular permalink (wp_oembed_add_discovery_links subset). Empty when home or
// permalink is blank.
func OEmbedDiscoveryLinks(home, permalink string) string {
	return OEmbedDiscoveryLinksWithHooks(nil, home, permalink)
}

// OEmbedDiscoveryLinksWithHooks builds discovery links and applies
// oembed_discovery_links (P3.92).
func OEmbedDiscoveryLinksWithHooks(h *Hooks, home, permalink string) string {
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	permalink = strings.TrimSpace(permalink)
	if home == "" || permalink == "" {
		return ""
	}
	jsonURL := GetOEmbedEndpointURLWithHooks(h, home, permalink, "json")
	xmlURL := GetOEmbedEndpointURLWithHooks(h, home, permalink, "xml")
	var b strings.Builder
	b.WriteString("<link rel=\"alternate\" title=\"oEmbed (JSON)\" type=\"application/json+oembed\" href=\"")
	b.WriteString(html.EscapeString(jsonURL))
	b.WriteString("\" />\n")
	b.WriteString("<link rel=\"alternate\" title=\"oEmbed (XML)\" type=\"text/xml+oembed\" href=\"")
	b.WriteString(html.EscapeString(xmlURL))
	b.WriteString("\" />\n")
	out := b.String()
	if h != nil {
		filtered := h.ApplyFilters("oembed_discovery_links", out)
		if s, ok := filtered.(string); ok {
			out = s
		}
	}
	return out
}
