package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveNetwork is the Tools → Network Setup stub (network.php, P7.96).
// Multisite / network admin is Tier 2 out of scope for GoPress; this screen
// explains that and links back to Tools.
func (h *Handler) serveNetwork(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	blogname := optionString(c, "blogname", "GoPress")
	canSetup := c != nil && c.CurrentUser != nil &&
		(c.UserCan(c.CurrentUser, "setup_network") || c.UserCan(c.CurrentUser, "manage_options"))

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Create a Network of WordPress Sites &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui network-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Create a Network of WordPress Sites</h1>\n")
	if !canSetup {
		b.WriteString("<div class=\"notice notice-error\"><p>Sorry, you are not allowed to manage options for this site.</p></div>\n")
	} else {
		b.WriteString("<div class=\"notice notice-info\" id=\"gopress-network-stub\"><p>")
		b.WriteString("Multisite network creation is <strong>out of scope</strong> for GoPress (Tier 2). ")
		b.WriteString("GoPress runs as a single site; <code>WP_ALLOW_MULTISITE</code> / Network Admin are not available.")
		b.WriteString("</p></div>\n")
		b.WriteString("<p>Use the <a href=\"/wp-admin/tools.php\">Tools</a> screen for site maintenance utilities.</p>\n")
	}
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Network", "disabled")
	writeHTML(w, http.StatusOK, b.String())
}
