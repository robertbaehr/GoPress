package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// servePluginInstall is the Plugins → Add Plugins stub (plugin-install.php, P7.19 / P7.114).
// Directory installs from wordpress.org remain out of scope; this screen gates on
// install_plugins (or activate_plugins) and points editors at the local Plugins screen.
func (h *Handler) servePluginInstall(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	blogname := optionString(c, "blogname", "GoPress")
	canInstall := c != nil && c.CurrentUser != nil &&
		(c.UserCan(c.CurrentUser, "install_plugins") || c.UserCan(c.CurrentUser, "activate_plugins"))

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>Add Plugins &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui plugin-install-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Add Plugins</h1>\n")
	if !canInstall {
		b.WriteString("<h1>You need a higher level of permission.</h1>\n")
		b.WriteString("<p>Sorry, you are not allowed to install plugins on this site.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Plugin-Install", "forbidden")
		writeHTML(w, http.StatusForbidden, b.String())
		return
	}
	b.WriteString("<div class=\"notice notice-info\" id=\"gopress-plugin-install-stub\"><p>")
	b.WriteString("Installing plugins from wordpress.org is <strong>out of scope</strong> for GoPress. ")
	b.WriteString("Plugin directory browse, upload ZIP, and remote install APIs are unavailable.")
	b.WriteString("</p></div>\n")
	b.WriteString("<p>Use the <a href=\"/wp-admin/plugins.php\">Plugins</a> screen to manage local plugins.</p>\n")
	b.WriteString("<div class=\"wp-filter plugin-install-php\" hidden>\n")
	b.WriteString("<div class=\"filter-count\"><span class=\"count plugin-count\">0</span></div>\n")
	b.WriteString("</div>\n")
	b.WriteString("<div class=\"plugin-browser content-filterable\"></div>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Plugin-Install", "disabled")
	// P7.114: reachable stub (200) for authorized users; remote install stays gone.
	writeHTML(w, http.StatusOK, b.String())
}
