package admin

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// adminChrome holds shared wp-admin header/footer markup (P7.15).
type adminChrome struct {
	Title     string
	BodyClass string
	Blogname  string
	Screen    string // e.g. dashboard, edit-php, profile
	HeadExtra string
	UserLogin string
}

func newAdminChrome(c *wp.Context, title, screen string) adminChrome {
	blogname := optionString(c, "blogname", "GoPress")
	login := ""
	if c != nil && c.CurrentUser != nil {
		login = c.CurrentUser.Login
	}
	return adminChrome{
		Title:     title,
		BodyClass: "wp-admin wp-core-ui no-js " + screen + " admin-bar admin-color-modern locale-en-us",
		Blogname:  blogname,
		Screen:    screen,
		UserLogin: login,
	}
}

func (ch adminChrome) openHTML() string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString("<html lang=\"en-US\">\n")
	b.WriteString("<head>\n")
	b.WriteString("<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" />\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(ch.Title))
	b.WriteString(" &lsaquo; ")
	b.WriteString(html.EscapeString(ch.Blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString(ch.HeadExtra)
	b.WriteString("<script>document.documentElement.className = document.documentElement.className.replace('no-js','js');</script>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body class=\"")
	b.WriteString(html.EscapeString(ch.BodyClass))
	b.WriteString("\">\n")
	b.WriteString(adminBarHTML(ch))
	b.WriteString("<div id=\"wpwrap\">\n")
	b.WriteString("<div id=\"adminmenumain\" role=\"navigation\" aria-label=\"Main menu\">\n")
	b.WriteString("<div id=\"adminmenuback\"></div>\n")
	b.WriteString("<div id=\"adminmenuwrap\">\n")
	b.WriteString("<ul id=\"adminmenu\">\n")
	b.WriteString(adminMenuHTML(ch.Screen))
	b.WriteString("</ul>\n</div>\n</div>\n")
	b.WriteString("<div id=\"wpcontent\">\n")
	b.WriteString("<div id=\"wpbody\" role=\"main\">\n")
	b.WriteString("<div id=\"wpbody-content\">\n")
	b.WriteString("<div class=\"wrap\">\n")
	return b.String()
}

func (ch adminChrome) closeHTML() string {
	var b strings.Builder
	b.WriteString("</div><!-- .wrap -->\n")
	b.WriteString("</div><!-- #wpbody-content -->\n")
	b.WriteString("</div><!-- #wpbody -->\n")
	b.WriteString("</div><!-- #wpcontent -->\n")
	b.WriteString("<div id=\"wpfooter\" role=\"contentinfo\">\n")
	b.WriteString("<p id=\"footer-left\" class=\"alignleft\"><span id=\"footer-thankyou\">Thank you for creating with <a href=\"https://wordpress.org/\">WordPress</a>.</span></p>\n")
	b.WriteString("<p id=\"footer-upgrade\" class=\"alignright\">GoPress admin stub</p>\n")
	b.WriteString("</div><!-- #wpfooter -->\n")
	b.WriteString("</div><!-- #wpwrap -->\n")
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

func adminBarHTML(ch adminChrome) string {
	var b strings.Builder
	b.WriteString("<div id=\"wpadminbar\" class=\"nojq nojs\">\n")
	b.WriteString("<div class=\"quicklinks\" id=\"wp-toolbar\" role=\"navigation\">\n")
	b.WriteString("<ul id=\"wp-admin-bar-root-default\" class=\"ab-top-menu\">\n")
	b.WriteString("<li id=\"wp-admin-bar-menu-toggle\"><a class=\"ab-item\" href=\"#\"><span class=\"ab-icon\"></span><span class=\"screen-reader-text\">Menu</span></a></li>\n")
	b.WriteString("<li id=\"wp-admin-bar-site-name\"><a class=\"ab-item\" href=\"/\">")
	b.WriteString(html.EscapeString(ch.Blogname))
	b.WriteString("</a></li>\n")
	b.WriteString("</ul>\n")
	b.WriteString("<ul id=\"wp-admin-bar-top-secondary\" class=\"ab-top-secondary ab-top-menu\">\n")
	b.WriteString("<li id=\"wp-admin-bar-my-account\"><a class=\"ab-item\" href=\"/wp-admin/profile.php\">")
	if ch.UserLogin != "" {
		b.WriteString("Howdy, ")
		b.WriteString(html.EscapeString(ch.UserLogin))
	} else {
		b.WriteString("Profile")
	}
	b.WriteString("</a></li>\n")
	b.WriteString("</ul>\n</div>\n</div>\n")
	return b.String()
}

func adminMenuHTML(screen string) string {
	items := []struct {
		id, href, label string
	}{
		{"menu-dashboard", "/wp-admin/", "Dashboard"},
		{"menu-posts", "/wp-admin/edit.php", "Posts"},
		{"menu-media", "/wp-admin/upload.php", "Media"},
		{"menu-pages", "/wp-admin/edit.php?post_type=page", "Pages"},
		{"menu-comments", "/wp-admin/edit-comments.php", "Comments"},
		{"menu-appearance", "/wp-admin/themes.php", "Appearance"},
		{"menu-plugins", "/wp-admin/plugins.php", "Plugins"},
		{"menu-users", "/wp-admin/users.php", "Users"},
		{"menu-tools", "/wp-admin/tools.php", "Tools"},
		{"menu-settings", "/wp-admin/options-general.php", "Settings"},
	}
	var b strings.Builder
	for _, it := range items {
		current := ""
		switch {
		case screen == "index-php" && it.id == "menu-dashboard":
			current = " current"
		case screen == "edit-php" && it.id == "menu-posts":
			current = " current"
		case screen == "plugins-php" && it.id == "menu-plugins":
			current = " current"
		case (screen == "options-general-php" || screen == "options-writing-php" || screen == "options-reading-php") && it.id == "menu-settings":
			current = " current"
		case screen == "profile-php" && it.id == "menu-users":
			current = " current"
		}
		b.WriteString("<li class=\"wp-has-submenu wp-not-current-submenu menu-top")
		b.WriteString(current)
		b.WriteString("\" id=\"")
		b.WriteString(it.id)
		b.WriteString("\"><a href=\"")
		b.WriteString(html.EscapeString(it.href))
		b.WriteString("\" class=\"wp-has-submenu wp-not-current-submenu menu-top")
		b.WriteString(current)
		b.WriteString("\"><div class=\"wp-menu-name\">")
		b.WriteString(html.EscapeString(it.label))
		b.WriteString("</div></a></li>\n")
	}
	return b.String()
}

// writeAdminPage emits a full wp-admin document with shared chrome around innerHTML.
func writeAdminPage(w http.ResponseWriter, c *wp.Context, ch adminChrome, innerHTML string) {
	writeAdminSettingsCookies(w, c)
	var b strings.Builder
	b.WriteString(ch.openHTML())
	b.WriteString(innerHTML)
	b.WriteString(ch.closeHTML())
	writeHTML(w, http.StatusOK, b.String())
}

// writeAdminSettingsCookies mirrors WP user settings cookies on admin screens.
func writeAdminSettingsCookies(w http.ResponseWriter, c *wp.Context) {
	uid := int64(0)
	if c != nil && c.CurrentUser != nil {
		uid = c.CurrentUser.ID
	}
	// WP deletes stale wp-settings-{uid} then refreshes wp-settings-time-{uid}.
	http.SetCookie(w, &http.Cookie{
		Name:    fmt.Sprintf("wp-settings-%d", uid),
		Value:   "deleted",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(1, 0).UTC(),
	})
	http.SetCookie(w, &http.Cookie{
		Name:   fmt.Sprintf("wp-settings-time-%d", uid),
		Value:  strconv.FormatInt(time.Now().Unix(), 10),
		Path:   "/",
		MaxAge: 31536000,
	})
}
