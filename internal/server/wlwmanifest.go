package server

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// wlwManifestStub is a minimal Windows Live Writer manifest (P3.28 / P3.134).
// Core removed the file in WP 6.3; GoPress keeps a compatibility stub so
// legacy clients probing /wp-includes/wlwmanifest.xml get XML, not a front 501.
const wlwManifestStub = `<?xml version="1.0" encoding="utf-8" ?>
<manifest xmlns="http://schemas.microsoft.com/wlw/manifest/weblog">
  <options>
    <clientType>WordPress</clientType>
    <supportsKeywords>Yes</supportsKeywords>
    <supportsGetTags>Yes</supportsGetTags>
  </options>
  <weblog>
    <serviceName>GoPress</serviceName>
    <homepageLinkText>View site</homepageLinkText>
    <adminLinkText>Dashboard</adminLinkText>
    <adminUrl><![CDATA[{blog-postapi-url}/../wp-admin/]]></adminUrl>
    <postEditingUrl><![CDATA[{blog-postapi-url}/../wp-admin/post.php?action=edit&post={post-id}]]></postEditingUrl>
  </weblog>
</manifest>
`

// serveWLWManifestStub serves the legacy WLW manifest path (P3.28 / P3.134).
// P3.134: emit rest_output_link_header + clear pingback discovery headers;
// front head never emits wlwmanifest_link (removed in WP 6.3).
func serveWLWManifestStub(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	// P3.134: clear X-Pingback / Link rel=pingback before discovery Link.
	wp.ClearPingbackHeaders(w)
	wp.RestOutputLinkHeader(w, c)
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	body := wlwManifestStub
	if home := wlwHomeURL(c); home != "" {
		// Optional: surface site home in a comment for legacy probes (body shape stable).
		_ = home
	}
	_, _ = w.Write([]byte(body))
}

func wlwHomeURL(c *wp.Context) string {
	if c == nil || c.Options == nil {
		return ""
	}
	siteURL, _ := c.Options.GetOption("siteurl", "").(string)
	home, _ := c.Options.GetOption("home", siteURL).(string)
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	if home == "" {
		home = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	}
	return home
}
