package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveUpdateCore is the Dashboard → Updates stub (update-core.php, P7.99).
// WordPress core updates remain out of scope; this screen explains that and
// gates on update_core / manage_options (replacing the P7.20 403 gone page).
// P7.167: pagenow/adminpage chrome, locale body class, admin_body_class.
func (h *Handler) serveUpdateCore(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	blogname := optionString(c, "blogname", "GoPress")
	canUpdate := c != nil && c.CurrentUser != nil &&
		(c.UserCan(c.CurrentUser, "update_core") || c.UserCan(c.CurrentUser, "manage_options"))

	bodyClass := "wp-admin wp-core-ui update-core-php locale-en-us"
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
	b.WriteString("<title>WordPress Updates &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<script>var ajaxurl = \"/wp-admin/admin-ajax.php\", pagenow = \"update-core\", typenow = \"\", adminpage = \"update-core-php\";</script>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body class=\"")
	b.WriteString(html.EscapeString(bodyClass))
	b.WriteString("\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>WordPress Updates</h1>\n")
	if !canUpdate {
		b.WriteString("<div class=\"notice notice-error\"><p>Sorry, you are not allowed to update Core for this site.</p></div>\n")
	} else {
		b.WriteString("<div class=\"notice notice-info\" id=\"gopress-update-core-stub\"><p>")
		b.WriteString("WordPress core updates via <code>update-core.php</code> are <strong>out of scope</strong> for GoPress. ")
		b.WriteString("Deploy a new GoPress build instead of upgrading through the admin UI.")
		b.WriteString("</p></div>\n")
		b.WriteString("<p>Use the <a href=\"/wp-admin/index.php\">Dashboard</a> for site overview.</p>\n")
	}
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Update-Core", "disabled")
	writeHTML(w, http.StatusOK, b.String())
}
