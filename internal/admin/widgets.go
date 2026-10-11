package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveWidgets is the Appearance → Widgets stub (widgets.php, P7.112).
// Classic/block widget admin and REST widgets/sidebars are out of scope;
// this screen gates on edit_theme_options and points editors at the Site Editor.
func (h *Handler) serveWidgets(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	blogname := optionString(c, "blogname", "GoPress")
	canEdit := c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "edit_theme_options")
	themeWidgets := wp.CurrentThemeSupports(c, "widgets") || wp.CurrentThemeSupports(c, "widgets-block-editor")

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<title>Widgets &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui widgets-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Widgets</h1>\n")
	if !canEdit {
		b.WriteString("<h1>You need a higher level of permission.</h1>\n")
		b.WriteString("<p>Sorry, you are not allowed to edit theme options on this site.</p>\n")
		b.WriteString("</div>\n</body>\n</html>\n")
		w.Header().Set("X-GoPress-Widgets", "forbidden")
		writeHTML(w, http.StatusForbidden, b.String())
		return
	}
	if !themeWidgets {
		b.WriteString("<div class=\"notice notice-warning\" id=\"gopress-widgets-no-support\"><p>")
		b.WriteString("The theme you are currently using is not widget-aware, meaning that it has no sidebars that you are able to change.")
		b.WriteString("</p></div>\n")
	}
	b.WriteString("<div class=\"notice notice-info\" id=\"gopress-widgets-stub\"><p>")
	b.WriteString("The Widgets screen (classic and block widget editor) is <strong>unavailable</strong> in GoPress. ")
	b.WriteString("Legacy widgets, sidebars, and the <code>/wp/v2/widgets</code> REST API are out of scope.")
	b.WriteString("</p></div>\n")
	b.WriteString("<p>Manage layout in the <a href=\"/wp-admin/site-editor.php\">Site Editor</a> ")
	b.WriteString("or switch themes from <a href=\"/wp-admin/themes.php\">Themes</a>.</p>\n")
	b.WriteString("<div id=\"widgets-editor\" class=\"blocks-widgets-container\" hidden></div>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-Widgets", "disabled")
	writeHTML(w, http.StatusOK, b.String())
}
