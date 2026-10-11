package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveThemeEditor is the Appearance → Theme File Editor stub (theme-editor.php, P7.37 / P7.115 / P7.156).
// edit_themes is denied by GoPress/oracle parity (map_meta_cap), matching WordPress admin
// menu gating — authorized users still receive 403. PHP theme file editing stays out of scope;
// the page points editors at the Site Editor / Themes screens.
// P7.156: theme/file/action query chrome, pagenow, updated notice, stylesheet header.
func (h *Handler) serveThemeEditor(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	stylesheet := optionString(c, "stylesheet", optionString(c, "template", "twentytwentyfive"))
	themeParam := ""
	fileParam := ""
	actionParam := ""
	updated := false
	if r != nil {
		q := r.URL.Query()
		themeParam = wp.SanitizeTextField(strings.TrimSpace(q.Get("theme")))
		fileParam = wp.SanitizeFileName(strings.TrimSpace(q.Get("file")))
		actionParam = wp.SanitizeKey(strings.TrimSpace(q.Get("action")))
		updated = q.Has("updated") // WP: ! empty( $_GET['updated'] ) style via presence
		if v := strings.TrimSpace(q.Get("updated")); v == "false" || v == "0" {
			updated = false
		}
	}
	if themeParam != "" {
		if key := wp.SanitizeKey(themeParam); key != "" {
			stylesheet = key
		}
	}
	// edit_themes is in deniedCaps; UserCan is false for all roles (parity with oracle).
	canEditThemes := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "edit_themes")

	bodyClass := "wp-admin wp-core-ui theme-editor-php locale-en-us"
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
	b.WriteString("<title>Edit Themes &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<script>var ajaxurl = \"/wp-admin/admin-ajax.php\", pagenow = \"theme-editor\", typenow = \"\", adminpage = \"theme-editor-php\";</script>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body class=\"")
	b.WriteString(html.EscapeString(bodyClass))
	b.WriteString("\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\"")
	b.WriteString(` data-theme="`)
	b.WriteString(html.EscapeString(stylesheet))
	b.WriteString(`"`)
	if fileParam != "" {
		b.WriteString(` data-file="`)
		b.WriteString(html.EscapeString(fileParam))
		b.WriteString(`"`)
	}
	if actionParam != "" {
		b.WriteString(` data-action="`)
		b.WriteString(html.EscapeString(actionParam))
		b.WriteString(`"`)
	}
	b.WriteString(">\n")
	b.WriteString("<h1>Theme File Editor</h1>\n")
	if updated {
		b.WriteString("<div id=\"message\" class=\"updated notice is-dismissible\"><p>File edited successfully.</p></div>\n")
	}
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
		if stylesheet != "" {
			w.Header().Set("X-GoPress-Theme-Editor-Theme", stylesheet)
		}
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
	if stylesheet != "" {
		w.Header().Set("X-GoPress-Theme-Editor-Theme", stylesheet)
	}
	writeHTML(w, http.StatusOK, b.String())
}
