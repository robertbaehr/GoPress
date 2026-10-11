package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveThemeEditor is the Appearance → Theme File Editor stub (theme-editor.php, P7.37 / P7.115).
// edit_themes is denied by GoPress/oracle parity (map_meta_cap), matching WordPress admin
// menu gating — authorized users still receive 403. PHP theme file editing stays out of scope;
// the page points editors at the Site Editor / Themes screens.
func (h *Handler) serveThemeEditor(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	blogname := optionString(c, "blogname", "GoPress")
	// edit_themes is in deniedCaps; UserCan is false for all roles (parity with oracle).
	canEditThemes := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "edit_themes")

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>Edit Themes &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui theme-editor-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Theme File Editor</h1>\n")
	if !canEditThemes {
		// Mirror WP admin.php / theme-editor.php capability denial (403).
		b.WriteString("<div class=\"notice notice-error\" id=\"gopress-theme-editor-stub\"><p>")
		b.WriteString("Sorry, you are not allowed to edit templates for this site.")
		b.WriteString("</p></div>\n")
		b.WriteString("<p>Editing PHP theme files via <code>theme-editor.php</code> is <strong>out of scope</strong> for GoPress. ")
		b.WriteString("Use the <a href=\"/wp-admin/site-editor.php\">Site Editor</a> ")
		b.WriteString("or the <a href=\"/wp-admin/themes.php\">Themes</a> screen instead.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Theme-Editor", "forbidden")
		writeHTML(w, http.StatusForbidden, b.String())
		return
	}
	// Unreachable under current deniedCaps; kept for future if file-edit caps reopen.
	b.WriteString("<div class=\"notice notice-info\" id=\"gopress-theme-editor-stub\"><p>")
	b.WriteString("Editing PHP theme files via <code>theme-editor.php</code> is <strong>out of scope</strong> for GoPress.")
	b.WriteString("</p></div>\n")
	b.WriteString("<p>Open the <a href=\"/wp-admin/site-editor.php\">Site Editor</a>.</p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Theme-Editor", "disabled")
	writeHTML(w, http.StatusOK, b.String())
}
