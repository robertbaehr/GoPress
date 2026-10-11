package admin

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// ensureWelcomePanelAction registers wp_welcome_panel on welcome_panel (P7.135).
func ensureWelcomePanelAction(c *wp.Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	if _, ok := c.Hooks.HasActionFunc("welcome_panel", writeWPWelcomePanel, 10); ok {
		return
	}
	c.Hooks.AddAction("welcome_panel", writeWPWelcomePanel, 10, 0)
}

func writeWPWelcomePanel(_ ...any) {
	// Content is assembled in welcomePanelInnerHTML; this hook marks presence
	// for has_action('welcome_panel') parity and allows remove_action.
}

// handleWelcomePanelDismiss processes ?welcome=0 (P7.135 / WP_Screen).
func handleWelcomePanelDismiss(w http.ResponseWriter, r *http.Request, c *wp.Context) bool {
	if r == nil || c == nil || c.CurrentUser == nil {
		return false
	}
	if r.URL.Query().Get("welcome") != "0" {
		return false
	}
	if !c.UserCan(c.CurrentUser, "edit_theme_options") {
		return false
	}
	_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, c.CurrentUser.ID, "show_welcome_panel", 0, "")
	http.Redirect(w, r, wp.AdminURL(c, ""), http.StatusFound)
	return true
}

// welcomePanelHTML builds the dashboard welcome panel (P7.135 / wp-admin/index.php).
func welcomePanelHTML(c *wp.Context) string {
	if c == nil || c.CurrentUser == nil || c.Hooks == nil {
		return ""
	}
	if !c.UserCan(c.CurrentUser, "edit_theme_options") {
		return ""
	}
	ensureWelcomePanelAction(c)
	if !c.Hooks.HasAction("welcome_panel") {
		return ""
	}
	option := showWelcomePanelOption(c)
	// 0 = hide, 1 = show, 2 = multisite owner (hide unless admin_email matches).
	hide := option == 0 || (option == 2 && !welcomePanelEmailMatches(c))
	classes := "welcome-panel"
	if hide {
		classes += " hidden"
	}
	nonce := c.CreateNonce("welcome-panel-nonce")
	dismiss := html.EscapeString(wp.AdminURL(c, "?welcome=0"))
	var b strings.Builder
	b.WriteString(`<div id="welcome-panel" class="`)
	b.WriteString(html.EscapeString(classes))
	b.WriteString(`">`)
	b.WriteString(`<input type="hidden" id="welcomepanelnonce" name="welcomepanelnonce" value="`)
	b.WriteString(html.EscapeString(nonce))
	b.WriteString(`" />`)
	b.WriteString(`<a class="welcome-panel-close" href="`)
	b.WriteString(dismiss)
	b.WriteString(`" aria-label="Dismiss the welcome panel">Dismiss</a>`)
	b.WriteString(welcomePanelInnerHTML(c))
	b.WriteString(`</div>`)
	// Fire action for plugin observers (content already rendered).
	c.Hooks.DoAction("welcome_panel")
	return b.String()
}

func showWelcomePanelOption(c *wp.Context) int {
	if c == nil || c.Meta == nil || c.CurrentUser == nil {
		return 1
	}
	raw := c.Meta.GetMetadata(wp.MetaUser, c.CurrentUser.ID, "show_welcome_panel", true)
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" || s == "<nil>" {
		return 1 // default show for single-site creators
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 1
	}
	return n
}

func welcomePanelEmailMatches(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil || c.Options == nil {
		return false
	}
	adminEmail := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", "")))
	return adminEmail != "" && strings.EqualFold(c.CurrentUser.Email, adminEmail)
}

// welcomePanelInnerHTML is wp_welcome_panel markup subset (P7.135).
func welcomePanelInnerHTML(c *wp.Context) string {
	ver := wp.Version
	if i := strings.IndexByte(ver, '-'); i >= 0 {
		ver = ver[:i]
	}
	isBlock := false
	if c != nil && c.Options != nil && c.Config != nil {
		stylesheet := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("stylesheet", "")))
		isBlock = theme.IsBlockTheme(c.Config.ABSPATH, stylesheet)
	}
	about := html.EscapeString(wp.AdminURL(c, "about.php"))
	var b strings.Builder
	b.WriteString(`<div class="welcome-panel-content">`)
	b.WriteString(`<div class="welcome-panel-header-wrap"><div class="welcome-panel-header">`)
	b.WriteString(`<h2>Welcome to WordPress!</h2>`)
	b.WriteString(`<p><a href="`)
	b.WriteString(about)
	b.WriteString(`">Learn more about the `)
	b.WriteString(html.EscapeString(ver))
	b.WriteString(` version.</a></p>`)
	b.WriteString(`</div></div>`)
	b.WriteString(`<div class="welcome-panel-column-container">`)
	b.WriteString(`<div class="welcome-panel-column"><div class="welcome-panel-column-content">`)
	b.WriteString(`<h3>Author rich content with blocks and patterns</h3>`)
	b.WriteString(`<p>Block patterns are pre-configured block layouts. Use them to get inspired or create new pages in a flash.</p>`)
	b.WriteString(`<a href="`)
	b.WriteString(html.EscapeString(wp.AdminURL(c, "post-new.php?post_type=page")))
	b.WriteString(`">Add a new page</a>`)
	b.WriteString(`</div></div>`)
	b.WriteString(`<div class="welcome-panel-column"><div class="welcome-panel-column-content">`)
	if isBlock {
		b.WriteString(`<h3>Customize your entire site with block themes</h3>`)
		b.WriteString(`<p>Design everything on your site &#8212; from the header down to the footer, all using blocks and patterns.</p>`)
		b.WriteString(`<a href="`)
		b.WriteString(html.EscapeString(wp.AdminURL(c, "site-editor.php")))
		b.WriteString(`">Open site editor</a>`)
	} else {
		b.WriteString(`<h3>Start Customizing</h3>`)
		b.WriteString(`<p>Configure your site&#8217;s logo, header, menus, and more in the Customizer.</p>`)
	}
	b.WriteString(`</div></div>`)
	b.WriteString(`<div class="welcome-panel-column"><div class="welcome-panel-column-content">`)
	if isBlock {
		b.WriteString(`<h3>Switch up your site&#8217;s look &amp; feel with Styles</h3>`)
		b.WriteString(`<p>Tweak your site, or give it a whole new look! Get creative &#8212; how about a new color palette or font?</p>`)
		b.WriteString(`<a href="`)
		b.WriteString(html.EscapeString(wp.AdminURL(c, "site-editor.php?p=/styles")))
		b.WriteString(`">Edit styles</a>`)
	} else {
		b.WriteString(`<h3>Discover a new way to build your site.</h3>`)
		b.WriteString(`<p>There is a new kind of WordPress theme, called a block theme, that lets you build the site you&#8217;ve always wanted &#8212; with blocks and styles.</p>`)
	}
	b.WriteString(`</div></div>`)
	b.WriteString(`</div></div>`)
	return b.String()
}

// ajaxUpdateWelcomePanel handles action=update-welcome-panel (P7.135).
func (h *Handler) ajaxUpdateWelcomePanel(w http.ResponseWriter, r *http.Request, c *wp.Context, loggedIn bool) {
	if !loggedIn || c == nil || c.CurrentUser == nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("0"))
		return
	}
	_ = r.ParseForm()
	nonce := strings.TrimSpace(r.FormValue("welcomepanelnonce"))
	if c.VerifyNonce(nonce, "welcome-panel-nonce") == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("-1"))
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_theme_options") {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("-1"))
		return
	}
	visible := 0
	if strings.TrimSpace(r.FormValue("visible")) != "" && r.FormValue("visible") != "0" {
		visible = 1
	}
	_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, c.CurrentUser.ID, "show_welcome_panel", visible, "")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("1"))
}
