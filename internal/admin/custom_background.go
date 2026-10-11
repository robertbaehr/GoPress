package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveCustomBackground is Appearance → Background (custom-background.php, P7.153).
// Gates on edit_theme_options; shows theme support status and theme_mod color/image;
// POST background_color with nonce persists via SetBackgroundColor.
func (h *Handler) serveCustomBackground(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	canEdit := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "edit_theme_options")
	themeBG := wp.CurrentThemeSupports(c, "custom-background")

	updated := false
	if canEdit && r != nil && r.Method == http.MethodPost {
		_ = r.ParseForm()
		action := r.FormValue("action")
		switch action {
		case "save", "custom-background":
			if c.VerifyNonce(r.FormValue("_wpnonce"), "custom-background") != 0 {
				color := r.FormValue("background-color")
				if color == "" {
					color = r.FormValue("background_color")
				}
				if err := wp.SetBackgroundColor(c, color); err == nil {
					updated = true
				}
				if img := strings.TrimSpace(r.FormValue("background-image")); img != "" {
					_ = wp.SetBackgroundImage(c, img)
					updated = true
				}
			}
		case "reset", "remove":
			nonceAction := "custom-background-reset"
			if action == "remove" {
				nonceAction = "custom-background-remove"
			}
			if c.VerifyNonce(r.FormValue("_wpnonce"), nonceAction) != 0 {
				_ = wp.SetBackgroundColor(c, "")
				_ = wp.SetBackgroundImage(c, "")
				updated = true
			}
		}
	}

	color := wp.GetBackgroundColor(c)
	image := wp.GetBackgroundImage(c)
	nonce := ""
	if c != nil {
		nonce = c.CreateNonce("custom-background")
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>Background &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui appearance_page_custom-background\">\n")
	b.WriteString(adminNoticesHTML(c))
	if updated {
		b.WriteString("<div id=\"message\" class=\"updated notice notice-success is-dismissible\"><p>Background updated.</p></div>\n")
	}
	b.WriteString("<div class=\"wrap\" id=\"custom-background\">\n")
	b.WriteString("<h1>Background</h1>\n")
	if !canEdit {
		b.WriteString("<h1>You need a higher level of permission.</h1>\n")
		b.WriteString("<p>Sorry, you are not allowed to edit theme options on this site.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Custom-Background", "forbidden")
		writeHTML(w, http.StatusForbidden, b.String())
		return
	}
	if !themeBG {
		b.WriteString("<div class=\"notice notice-warning\" id=\"gopress-custom-background-no-support\"><p>")
		b.WriteString("The current theme does not support custom backgrounds. ")
		b.WriteString("You can still preview stored theme mods below.")
		b.WriteString("</p></div>\n")
	}
	b.WriteString("<div class=\"notice notice-info\" id=\"gopress-custom-background-stub\"><p>")
	b.WriteString("GoPress custom background (P7.153): color theme_mod save is supported; ")
	b.WriteString("image upload UI remains a stub.")
	b.WriteString("</p></div>\n")

	b.WriteString("<form method=\"post\" action=\"/wp-admin/custom-background.php\">\n")
	b.WriteString("<input type=\"hidden\" name=\"action\" value=\"save\" />\n")
	b.WriteString("<input type=\"hidden\" name=\"_wpnonce\" value=\"")
	b.WriteString(html.EscapeString(nonce))
	b.WriteString("\" />\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr><th scope=\"row\"><label for=\"background-color\">Background Color</label></th>")
	b.WriteString("<td><input name=\"background-color\" type=\"text\" id=\"background-color\" value=\"")
	b.WriteString(html.EscapeString(color))
	b.WriteString("\" class=\"regular-text\" />")
	if color != "" {
		b.WriteString(` <span class="description">#`)
		b.WriteString(html.EscapeString(color))
		b.WriteString("</span>")
	}
	b.WriteString("</td></tr>\n")
	b.WriteString("<tr><th scope=\"row\">Background Image</th><td>")
	if image != "" {
		b.WriteString(`<div id="custom-background-image"><img class="custom-background-image" src="`)
		b.WriteString(html.EscapeString(image))
		b.WriteString(`" alt="" /></div>`)
	} else {
		b.WriteString("<p class=\"description\">No background image set.</p>")
	}
	b.WriteString("</td></tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"save-background-options\" id=\"save-background-options\" class=\"button button-primary\" value=\"Save Changes\" /></p>\n")
	b.WriteString("</form>\n")
	b.WriteString("<p><a href=\"/wp-admin/themes.php\">&larr; Themes</a></p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Custom-Background", "ok")
	if color != "" {
		w.Header().Set("X-GoPress-Background-Color", color)
	}
	writeHTML(w, http.StatusOK, b.String())
}
