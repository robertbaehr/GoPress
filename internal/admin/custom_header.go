package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveCustomHeader is Appearance → Header (custom-header.php, P7.154).
// Gates on edit_theme_options; shows theme support status and theme_mod image/text color;
// POST header-text-color / header-image with nonce persists via SetHeader*.
func (h *Handler) serveCustomHeader(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	canEdit := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "edit_theme_options")
	themeHeader := wp.CurrentThemeSupports(c, "custom-header")

	updated := false
	if canEdit && r != nil && r.Method == http.MethodPost {
		_ = r.ParseForm()
		action := r.FormValue("action")
		switch action {
		case "save", "custom-header":
			if c.VerifyNonce(r.FormValue("_wpnonce"), "custom-header") != 0 {
				color := r.FormValue("header-text-color")
				if color == "" {
					color = r.FormValue("header_textcolor")
				}
				if err := wp.SetHeaderTextColor(c, color); err == nil {
					updated = true
				}
				if img := strings.TrimSpace(r.FormValue("header-image")); img != "" {
					_ = wp.SetHeaderImage(c, img)
					updated = true
				}
			}
		case "reset", "remove":
			nonceAction := "custom-header-reset"
			if action == "remove" {
				nonceAction = "custom-header-remove"
			}
			if c.VerifyNonce(r.FormValue("_wpnonce"), nonceAction) != 0 {
				_ = wp.SetHeaderTextColor(c, "")
				_ = wp.SetHeaderImage(c, "")
				updated = true
			}
		}
	}

	color := wp.GetHeaderTextColor(c)
	image := wp.GetHeaderImage(c)
	nonce := ""
	if c != nil {
		nonce = c.CreateNonce("custom-header")
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>Header &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui appearance_page_custom-header\">\n")
	b.WriteString(adminNoticesHTML(c))
	if updated {
		b.WriteString("<div id=\"message\" class=\"updated notice notice-success is-dismissible\"><p>Header updated.</p></div>\n")
	}
	b.WriteString("<div class=\"wrap\" id=\"custom-header\">\n")
	b.WriteString("<h1>Header</h1>\n")
	if !canEdit {
		b.WriteString("<h1>You need a higher level of permission.</h1>\n")
		b.WriteString("<p>Sorry, you are not allowed to edit theme options on this site.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Custom-Header", "forbidden")
		writeHTML(w, http.StatusForbidden, b.String())
		return
	}
	if !themeHeader {
		b.WriteString("<div class=\"notice notice-warning\" id=\"gopress-custom-header-no-support\"><p>")
		b.WriteString("The current theme does not support custom headers. ")
		b.WriteString("You can still preview stored theme mods below.")
		b.WriteString("</p></div>\n")
	}
	b.WriteString("<div class=\"notice notice-info\" id=\"gopress-custom-header-stub\"><p>")
	b.WriteString("GoPress custom header (P7.154): text color theme_mod save is supported; ")
	b.WriteString("image upload UI remains a stub.")
	b.WriteString("</p></div>\n")

	b.WriteString("<form method=\"post\" action=\"/wp-admin/custom-header.php\">\n")
	b.WriteString("<input type=\"hidden\" name=\"action\" value=\"save\" />\n")
	b.WriteString("<input type=\"hidden\" name=\"_wpnonce\" value=\"")
	b.WriteString(html.EscapeString(nonce))
	b.WriteString("\" />\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr><th scope=\"row\"><label for=\"header-text-color\">Header Text Color</label></th>")
	b.WriteString("<td><input name=\"header-text-color\" type=\"text\" id=\"header-text-color\" value=\"")
	b.WriteString(html.EscapeString(color))
	b.WriteString("\" class=\"regular-text\" />")
	if color != "" {
		b.WriteString(` <span class="description">#`)
		b.WriteString(html.EscapeString(color))
		b.WriteString("</span>")
	}
	b.WriteString("</td></tr>\n")
	b.WriteString("<tr><th scope=\"row\">Header Image</th><td>")
	if image != "" {
		b.WriteString(`<div id="custom-header-image"><img class="custom-header-image" src="`)
		b.WriteString(html.EscapeString(image))
		b.WriteString(`" alt="" /></div>`)
	} else {
		b.WriteString("<p class=\"description\">No header image set.</p>")
	}
	b.WriteString("</td></tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"save-header-options\" id=\"save-header-options\" class=\"button button-primary\" value=\"Save Changes\" /></p>\n")
	b.WriteString("</form>\n")
	b.WriteString("<p><a href=\"/wp-admin/themes.php\">&larr; Themes</a></p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Custom-Header", "ok")
	if color != "" {
		w.Header().Set("X-GoPress-Header-Text-Color", color)
	}
	writeHTML(w, http.StatusOK, b.String())
}
