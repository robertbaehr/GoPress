package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveThemeInstall is the Appearance → Add Themes stub (theme-install.php, P7.18 / P7.113).
// Directory installs from wordpress.org remain out of scope; this screen gates on
// install_themes (or switch_themes) and points editors at the local Themes screen.
// P7.168: pagenow/adminpage chrome, locale body class, admin_body_class.
func (h *Handler) serveThemeInstall(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	blogname := optionString(c, "blogname", "GoPress")
	canInstall := c != nil && c.CurrentUser != nil &&
		(c.UserCan(c.CurrentUser, "install_themes") || c.UserCan(c.CurrentUser, "switch_themes"))

	bodyClass := "wp-admin wp-core-ui theme-install-php locale-en-us"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("admin_body_class", ""); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				bodyClass += " " + strings.TrimSpace(s)
			}
		}
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>Add Themes &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<script>var ajaxurl = \"/wp-admin/admin-ajax.php\", pagenow = \"theme-install\", typenow = \"\", adminpage = \"theme-install-php\";</script>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body class=\"")
	b.WriteString(html.EscapeString(bodyClass))
	b.WriteString("\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Add Themes</h1>\n")
	if !canInstall {
		b.WriteString("<h1>You need a higher level of permission.</h1>\n")
		b.WriteString("<p>Sorry, you are not allowed to install themes on this site.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Theme-Install", "forbidden")
		writeHTML(w, http.StatusForbidden, b.String())
		return
	}
	b.WriteString("<div class=\"notice notice-info\" id=\"gopress-theme-install-stub\"><p>")
	b.WriteString("Installing themes from wordpress.org is <strong>out of scope</strong> for GoPress. ")
	b.WriteString("Theme directory browse, upload ZIP, and remote install APIs are unavailable.")
	b.WriteString("</p></div>\n")
	b.WriteString("<p>Use the <a href=\"/wp-admin/themes.php\">Themes</a> screen to manage local themes, ")
	b.WriteString("or open the <a href=\"/wp-admin/site-editor.php\">Site Editor</a>.</p>\n")
	b.WriteString("<div class=\"wp-filter theme-install-php\" hidden>\n")
	b.WriteString("<div class=\"filter-count\"><span class=\"count theme-count\">0</span></div>\n")
	b.WriteString("</div>\n")
	b.WriteString("<div class=\"theme-browser content-filterable\"></div>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Theme-Install", "disabled")
	// P7.113: reachable stub (200) for authorized users; remote install stays gone.
	writeHTML(w, http.StatusOK, b.String())
}
