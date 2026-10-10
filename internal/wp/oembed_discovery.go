// oEmbed discovery links (P3.20) — subset of wp_oembed_add_discovery_links.
package wp

import (
	"html"
	"net/url"
	"strings"
)

// OEmbedDiscoveryLinks returns the JSON/XML oEmbed alternate <link> tags for a
// singular permalink (wp_oembed_add_discovery_links subset). Empty when home or
// permalink is blank.
func OEmbedDiscoveryLinks(home, permalink string) string {
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	permalink = strings.TrimSpace(permalink)
	if home == "" || permalink == "" {
		return ""
	}
	endpoint := home + "/wp-json/oembed/1.0/embed?url=" + url.QueryEscape(permalink)
	var b strings.Builder
	b.WriteString("<link rel=\"alternate\" title=\"oEmbed (JSON)\" type=\"application/json+oembed\" href=\"")
	b.WriteString(html.EscapeString(endpoint))
	b.WriteString("\" />\n")
	b.WriteString("<link rel=\"alternate\" title=\"oEmbed (XML)\" type=\"text/xml+oembed\" href=\"")
	b.WriteString(html.EscapeString(endpoint + "&format=xml"))
	b.WriteString("\" />\n")
	return b.String()
}
