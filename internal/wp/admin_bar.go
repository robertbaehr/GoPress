// Admin bar stub (P6.5) — minimal toolbar HTML for logged-in front requests.
package wp

import (
	"html"
	"strings"
)

// AdminBarHTML returns minimal #wpadminbar markup when a user is logged in.
// Empty string when there is no current user.
func AdminBarHTML(c *Context) string {
	if c == nil || c.CurrentUser == nil {
		return ""
	}
	blogname := optionStringValue(c, "blogname", "GoPress")
	login := c.CurrentUser.Login
	var b strings.Builder
	b.WriteString(`<div id="wpadminbar" class="nojq nojs" role="navigation" aria-label="Toolbar">`)
	b.WriteString(`<div class="quicklinks" id="wp-toolbar" role="navigation">`)
	b.WriteString(`<ul id="wp-admin-bar-root-default" class="ab-top-menu">`)
	b.WriteString(`<li id="wp-admin-bar-site-name" class="menupop"><a class="ab-item" href="/wp-admin/">`)
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(`</a></li>`)
	b.WriteString(`<li id="wp-admin-bar-new-content" class="menupop"><a class="ab-item" href="/wp-admin/post-new.php">New</a></li>`)
	b.WriteString(`</ul>`)
	b.WriteString(`<ul id="wp-admin-bar-top-secondary" class="ab-top-secondary ab-top-menu">`)
	b.WriteString(`<li id="wp-admin-bar-my-account" class="menupop with-avatar"><a class="ab-item" href="/wp-admin/profile.php">Howdy, `)
	b.WriteString(html.EscapeString(login))
	b.WriteString(`</a></li>`)
	b.WriteString(`</ul></div></div>`)
	return b.String()
}

// BodyClassWithAdminBar appends admin-bar to a body class list when logged in.
func BodyClassWithAdminBar(c *Context, classes string) string {
	if c == nil || c.CurrentUser == nil {
		return classes
	}
	classes = strings.TrimSpace(classes)
	if classes == "" {
		return "admin-bar"
	}
	if strings.Contains(" "+classes+" ", " admin-bar ") {
		return classes
	}
	return classes + " admin-bar"
}
