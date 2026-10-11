package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// servePluginEditor is the Plugins → Plugin File Editor stub (plugin-editor.php, P7.38 / P7.116 / P7.157).
// edit_plugins is denied by GoPress/oracle parity (map_meta_cap), matching WordPress admin
// menu gating — authorized users still receive 403. PHP plugin file editing stays out of scope;
// the page points editors at the Plugins screen.
// P7.157: plugin/file/action query chrome, pagenow, updated notice.
func (h *Handler) servePluginEditor(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	pluginParam := ""
	fileParam := ""
	actionParam := ""
	updated := false
	if r != nil {
		q := r.URL.Query()
		pluginParam = wp.SanitizeTextField(strings.TrimSpace(q.Get("plugin")))
		fileParam = wp.SanitizeFileName(strings.TrimSpace(q.Get("file")))
		actionParam = wp.SanitizeKey(strings.TrimSpace(q.Get("action")))
		updated = q.Has("updated")
		if v := strings.TrimSpace(q.Get("updated")); v == "false" || v == "0" {
			updated = false
		}
	}
	// edit_plugins is in deniedCaps; UserCan is false for all roles (parity with oracle).
	canEditPlugins := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "edit_plugins")

	bodyClass := "wp-admin wp-core-ui plugin-editor-php locale-en-us"
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
	b.WriteString("<title>Edit Plugins &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<script>var ajaxurl = \"/wp-admin/admin-ajax.php\", pagenow = \"plugin-editor\", typenow = \"\", adminpage = \"plugin-editor-php\";</script>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body class=\"")
	b.WriteString(html.EscapeString(bodyClass))
	b.WriteString("\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\"")
	if pluginParam != "" {
		b.WriteString(` data-plugin="`)
		b.WriteString(html.EscapeString(pluginParam))
		b.WriteString(`"`)
	}
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
	b.WriteString("<h1>Plugin File Editor</h1>\n")
	if updated {
		b.WriteString("<div id=\"message\" class=\"updated notice is-dismissible\"><p>File edited successfully.</p></div>\n")
	}
	if !canEditPlugins {
		// Mirror WP admin.php / plugin-editor.php capability denial (403).
		b.WriteString("<div class=\"notice notice-error\" id=\"gopress-plugin-editor-stub\"><p>")
		b.WriteString("Sorry, you are not allowed to edit plugins for this site.")
		b.WriteString("</p></div>\n")
		b.WriteString("<p>Editing PHP plugin files via <code>plugin-editor.php</code> is <strong>out of scope</strong> for GoPress. ")
		b.WriteString("Use the <a href=\"/wp-admin/plugins.php\">Plugins</a> screen to manage local plugins.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Plugin-Editor", "forbidden")
		if pluginParam != "" {
			w.Header().Set("X-GoPress-Plugin-Editor-Plugin", pluginParam)
		}
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
	if pluginParam != "" {
		w.Header().Set("X-GoPress-Plugin-Editor-Plugin", pluginParam)
	}
	writeHTML(w, http.StatusOK, b.String())
}
