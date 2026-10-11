package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveLinkManager is the Links Manager stub (link-manager.php / link-add.php / link.php, P7.25 / P7.118).
// manage_links is never granted (link_manager_enabled permanently off); authorized users still receive 403.
// Classic bookmarks UI stays out of scope; the page points editors at the Dashboard.
func (h *Handler) serveLinkManager(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	path := ""
	if r != nil {
		path = r.URL.Path
	}
	blogname := optionString(c, "blogname", "GoPress")
	canManage := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "manage_links")

	title := "Links"
	switch path {
	case "/wp-admin/link-add.php":
		title = "Add Link"
	case "/wp-admin/link.php":
		title = "Edit Link"
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString(" &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui link-manager-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n")
	if !canManage {
		b.WriteString("<div class=\"notice notice-error\" id=\"gopress-link-manager-stub\"><p>")
		b.WriteString("Sorry, you are not allowed to edit the links for this site.")
		b.WriteString("</p></div>\n")
		b.WriteString("<p>The classic Links Manager (<code>link-manager.php</code>) is <strong>out of scope</strong> for GoPress. ")
		b.WriteString("<code>link_manager_enabled</code> is permanently off. ")
		b.WriteString("Use the <a href=\"/wp-admin/\">Dashboard</a> instead.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Link-Manager", "forbidden")
		writeHTML(w, http.StatusForbidden, b.String())
		return
	}
	// Unreachable under current link_manager_enabled=0; kept for future if Links reopen.
	b.WriteString("<div class=\"notice notice-info\" id=\"gopress-link-manager-stub\"><p>")
	b.WriteString("The classic Links Manager is <strong>out of scope</strong> for GoPress.")
	b.WriteString("</p></div>\n")
	b.WriteString("<p>Open the <a href=\"/wp-admin/\">Dashboard</a>.</p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Link-Manager", "disabled")
	writeHTML(w, http.StatusOK, b.String())
}
