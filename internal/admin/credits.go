package admin

import (
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveAbout is the About → What's New screen stub (P7.95 / P7.158).
// P7.158: pagenow/adminpage chrome, locale body class, admin_body_class,
// data-version wrap attr, X-GoPress-About-Version header.
func (h *Handler) serveAbout(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	displayVersion := strings.SplitN(wp.Version, "-", 2)[0]
	displayMajor := "7.1"
	if parts := strings.SplitN(displayVersion, ".", 3); len(parts) >= 2 {
		displayMajor = parts[0] + "." + parts[1]
	}
	tabParam := ""
	if r != nil {
		tabParam = wp.SanitizeKey(strings.TrimSpace(r.URL.Query().Get("tab")))
	}
	bodyClass := "wp-admin wp-core-ui about-php locale-en-us"
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
	b.WriteString("<title>About &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<script>var ajaxurl = \"/wp-admin/admin-ajax.php\", pagenow = \"about\", typenow = \"\", adminpage = \"about-php\";</script>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body class=\"")
	b.WriteString(html.EscapeString(bodyClass))
	b.WriteString("\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap about__container\"")
	b.WriteString(` data-version="`)
	b.WriteString(html.EscapeString(displayVersion))
	b.WriteString(`"`)
	if tabParam != "" {
		b.WriteString(` data-tab="`)
		b.WriteString(html.EscapeString(tabParam))
		b.WriteString(`"`)
	}
	b.WriteString(">\n")
	b.WriteString("<div class=\"about__header\">\n")
	b.WriteString("<div class=\"about__header-title\"><h1>WordPress ")
	b.WriteString(html.EscapeString(displayVersion))
	b.WriteString("</h1></div>\n")
	b.WriteString("</div>\n")
	b.WriteString(aboutTabsHTML("about"))
	b.WriteString("<div class=\"about__section\">\n")
	b.WriteString("<div class=\"column is-left-padding-zero is-right-padding-zero\">\n")
	b.WriteString("<h2>Welcome to WordPress ")
	b.WriteString(html.EscapeString(displayMajor))
	b.WriteString("</h2>\n")
	b.WriteString("<p class=\"is-subheading\">GoPress about.php stub (P7.95) — release feature copy lands later.</p>\n")
	b.WriteString("</div>\n</div>\n")
	b.WriteString("<hr class=\"is-large\" />\n")
	b.WriteString("<div class=\"return-to-dashboard\">\n")
	b.WriteString("<a href=\"index.php\">Go to Dashboard &rarr; Home</a>\n")
	b.WriteString("</div>\n</div>\n</body>\n</html>\n")
	w.Header().Set("X-GoPress-About-Version", displayVersion)
	writeHTML(w, http.StatusOK, b.String())
}

// serveCredits is the About → Credits screen stub (P7.94).
func (h *Handler) serveCredits(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Credits &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui about-php credits-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap about__container\">\n")
	b.WriteString("<div class=\"about__header\">\n")
	b.WriteString("<div class=\"about__header-title\"><h1>Contributors</h1></div>\n")
	b.WriteString("<div class=\"about__header-text\">Created by a worldwide team of passionate individuals</div>\n")
	b.WriteString("</div>\n")
	b.WriteString(aboutTabsHTML("credits"))
	b.WriteString("<div class=\"about__section has-1-column has-gutters\">\n")
	b.WriteString("<div class=\"column aligncenter\">\n")
	b.WriteString("<p>WordPress is created by a <a href=\"https://wordpress.org/about/\">worldwide team</a> of passionate individuals.<br />\n")
	b.WriteString("<a href=\"https://make.wordpress.org/contribute/\">Get involved in WordPress.</a></p>\n")
	b.WriteString("<p class=\"description\">GoPress credits stub (P7.94) — remote wp_credits() lists land later.</p>\n")
	b.WriteString("</div>\n</div>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveFreedoms is the About → Freedoms screen stub (P7.94).
func (h *Handler) serveFreedoms(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// WP 5.3+: freedoms.php?privacy-notice redirects to privacy.php.
	if _, ok := r.URL.Query()["privacy-notice"]; ok {
		http.Redirect(w, r, "/wp-admin/options-privacy.php", http.StatusMovedPermanently)
		return
	}
	blogname := optionString(c, "blogname", "GoPress")
	pluginsURL := "https://wordpress.org/plugins/"
	themesURL := "https://wordpress.org/themes/"
	if c != nil && c.CurrentUser != nil {
		if c.UserCan(c.CurrentUser, "activate_plugins") {
			pluginsURL = "/wp-admin/plugins.php"
		}
		if c.UserCan(c.CurrentUser, "switch_themes") {
			themesURL = "/wp-admin/themes.php"
		}
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Freedoms &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui about-php freedoms-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap about__container\">\n")
	b.WriteString("<div class=\"about__header\">\n")
	b.WriteString("<div class=\"about__header-title\"><h1>The Four Freedoms</h1></div>\n")
	b.WriteString("<div class=\"about__header-text\">WordPress is free and open source software</div>\n")
	b.WriteString("</div>\n")
	b.WriteString(aboutTabsHTML("freedoms"))
	b.WriteString("<div class=\"about__section is-feature\">\n")
	b.WriteString("<p class=\"about-description\">WordPress comes with some awesome, worldview-changing rights courtesy of its ")
	b.WriteString("<a href=\"https://wordpress.org/about/license/\">license</a>, the GPL.</p>\n")
	b.WriteString("</div>\n")
	b.WriteString("<div class=\"about__section has-2-columns\">\n")
	freedoms := []struct{ title, body string }{
		{"The 1st Freedom", "To run the program for any purpose."},
		{"The 2nd Freedom", "To study how the program works and change it to make it do what you wish."},
		{"The 3rd Freedom", "To redistribute."},
		{"The 4th Freedom", "To distribute copies of your modified versions to others."},
	}
	for _, f := range freedoms {
		b.WriteString("<div class=\"column aligncenter\">\n")
		b.WriteString("<h2 class=\"is-smaller-heading\">")
		b.WriteString(html.EscapeString(f.title))
		b.WriteString("</h2>\n<p>")
		b.WriteString(html.EscapeString(f.body))
		b.WriteString("</p>\n</div>\n")
	}
	b.WriteString("</div>\n")
	b.WriteString("<div class=\"about__section has-1-column\">\n<div class=\"column\">\n")
	b.WriteString("<p>Every plugin and theme in WordPress.org&#8217;s directory is 100% GPL or a similarly free and compatible license, so you can feel safe finding ")
	b.WriteString("<a href=\"")
	b.WriteString(html.EscapeString(pluginsURL))
	b.WriteString("\">plugins</a> and <a href=\"")
	b.WriteString(html.EscapeString(themesURL))
	b.WriteString("\">themes</a> there.</p>\n")
	b.WriteString("<p class=\"description\">GoPress freedoms stub (P7.94).</p>\n")
	b.WriteString("</div>\n</div>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// aboutTabsHTML emits About secondary nav tabs (credits/freedoms active) (P7.94).
func aboutTabsHTML(active string) string {
	tabs := []struct{ slug, href, label string }{
		{"about", "about.php", "What&#8217;s New"},
		{"credits", "credits.php", "Credits"},
		{"freedoms", "freedoms.php", "Freedoms"},
		{"privacy", "options-privacy.php", "Privacy"},
		{"contribute", "https://make.wordpress.org/contribute/", "Get Involved"},
	}
	var b strings.Builder
	b.WriteString(`<nav class="about__header-navigation nav-tab-wrapper wp-clearfix" aria-label="Secondary menu">`)
	for _, tab := range tabs {
		class := "nav-tab"
		aria := ""
		if tab.slug == active {
			class += " nav-tab-active"
			aria = ` aria-current="page"`
		}
		b.WriteString(`<a href="`)
		b.WriteString(tab.href)
		b.WriteString(`" class="`)
		b.WriteString(class)
		b.WriteString(`"`)
		b.WriteString(aria)
		b.WriteString(`>`)
		b.WriteString(tab.label)
		b.WriteString(`</a>`)
	}
	b.WriteString("</nav>\n")
	return b.String()
}
