package server

import (
	"net/http"
)

// wlwManifestStub is a minimal Windows Live Writer manifest (P3.28).
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

// serveWLWManifestStub serves the legacy WLW manifest path (P3.28).
func serveWLWManifestStub(w http.ResponseWriter, r *http.Request) {
	_ = r
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(wlwManifestStub))
}
