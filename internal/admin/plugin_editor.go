package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// servePluginEditor is the Plugins → Plugin File Editor stub (plugin-editor.php, P7.38 / P7.116).
// edit_plugins is denied by GoPress/oracle parity (map_meta_cap), matching WordPress admin
// menu gating — authorized users still receive 403. PHP plugin file editing stays out of scope;
// the page points editors at the Plugins screen.
func (h *Handler) servePluginEditor(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	blogname := optionString(c, "blogname", "GoPress")
	// edit_plugins is in deniedCaps; UserCan is false for all roles (parity with oracle).
	canEditPlugins := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "edit_plugins")

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>Edit Plugins &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui plugin-editor-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Plugin File Editor</h1>\n")
	if !canEditPlugins {
		// Mirror WP admin.php / plugin-editor.php capability denial (403).
		b.WriteString("<div class=\"notice notice-error\" id=\"gopress-plugin-editor-stub\"><p>")
		b.WriteString("Sorry, you are not allowed to edit plugins for this site.")
		b.WriteString("</p></div>\n")
		b.WriteString("<p>Editing PHP plugin files via <code>plugin-editor.php</code> is <strong>out of scope</strong> for GoPress. ")
		b.WriteString("Use the <a href=\"/wp-admin/plugins.php\">Plugins</a> screen to manage local plugins.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Plugin-Editor", "forbidden")
		writeHTML(w, http.StatusForbidden, b.String())
		return
	}
	// Unreachable under current deniedCaps; kept for future if file-edit caps reopen.
	b.WriteString("<div class=\"notice notice-info\" id=\"gopress-plugin-editor-stub\"><p>")
	b.WriteString("Editing PHP plugin files via <code>plugin-editor.php</code> is <strong>out of scope</strong> for GoPress.")
	b.WriteString("</p></div>\n")
	b.WriteString("<p>Open the <a href=\"/wp-admin/plugins.php\">Plugins</a> screen.</p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Plugin-Editor", "disabled")
	writeHTML(w, http.StatusOK, b.String())
}
