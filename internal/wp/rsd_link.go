package wp

// RsdLink returns the EditURI RSD discovery link for wp_head (rsd_link, P3.135).
// Matches core: site_url( 'xmlrpc.php?rsd', 'rpc' ) via esc_url.
func RsdLink(c *Context) string {
	href := SiteURL(c, "xmlrpc.php?rsd", "rpc")
	return `<link rel="EditURI" type="application/rsd+xml" title="RSD" href="` + EscURL(href) + `" />` + "\n"
}
