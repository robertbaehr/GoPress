// Admin bar stub (P6.5 / P7.128) — toolbar HTML for logged-in front requests.
package wp

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// ShowAdminBar sets the show_admin_bar preference for the request (show_admin_bar(), P7.128).
func ShowAdminBar(c *Context, show bool) {
	if c == nil || c.Scope == nil {
		return
	}
	c.Scope.ShowAdminBar = &show
}

// IsAdminBarShowing mirrors is_admin_bar_showing() (P7.128).
// False for ajax/REST/JSON/login/embed; otherwise logged-in front preference + filter.
func IsAdminBarShowing(c *Context) bool {
	if c == nil {
		return false
	}
	if c.Scope != nil {
		if c.Scope.IsAjax || c.Scope.IsREST || c.Scope.IsLogin || c.Scope.IsEmbed {
			return false
		}
		if c.Scope.IsAdmin {
			return true
		}
	}
	if c.Req != nil && IsJSONRequest(c.Req) {
		return false
	}

	show := false
	if c.Scope != nil && c.Scope.ShowAdminBar != nil {
		show = *c.Scope.ShowAdminBar
	} else if c.CurrentUser != nil {
		// Default preference true when unset (_get_admin_bar_pref).
		show = adminBarUserPref(c)
	}

	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("show_admin_bar", show)
		switch v := filtered.(type) {
		case bool:
			show = v
		case int:
			show = v != 0
		case string:
			s := strings.ToLower(strings.TrimSpace(v))
			show = s == "1" || s == "true" || s == "yes" || s == "on"
		}
	}
	return show
}

func adminBarUserPref(c *Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	// get_user_option('show_admin_bar_front'): false/missing → true; 'true' → true.
	if c.Meta == nil {
		return true
	}
	raw := c.Meta.GetMetadata(MetaUser, c.CurrentUser.ID, "show_admin_bar_front", true)
	switch v := raw.(type) {
	case string:
		if v == "" {
			return true
		}
		return v == "true" || v == "1"
	case bool:
		return v
	default:
		return true
	}
}

// AdminBarHTML returns #wpadminbar markup when the admin bar should show (P6.5 / P7.128).
// Empty string when hidden or anonymous.
func AdminBarHTML(c *Context) string {
	if !IsAdminBarShowing(c) || c == nil || c.CurrentUser == nil {
		return ""
	}
	blogname := optionStringValue(c, "blogname", "GoPress")
	login := c.CurrentUser.Login
	home := strings.TrimRight(HomeURL(c, "/", ""), "/")
	if home == "" {
		home = ""
	}
	var b strings.Builder
	b.WriteString(`<div id="wpadminbar" class="nojq nojs" role="navigation" aria-label="Toolbar">`)
	b.WriteString(`<div class="quicklinks" id="wp-toolbar" role="navigation">`)
	b.WriteString(`<ul id="wp-admin-bar-root-default" class="ab-top-menu">`)
	b.WriteString(`<li id="wp-admin-bar-site-name" class="menupop"><a class="ab-item" href="/wp-admin/">`)
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(`</a></li>`)
	if c.CurrentUserCan("edit_posts") {
		b.WriteString(`<li id="wp-admin-bar-comments"><a class="ab-item" href="/wp-admin/edit-comments.php">Comments</a></li>`)
		b.WriteString(`<li id="wp-admin-bar-new-content" class="menupop"><a class="ab-item" href="/wp-admin/post-new.php">New</a></li>`)
	}
	// P7.128: Edit link when viewing a singular post the user can edit.
	if editURL := adminBarEditURL(c); editURL != "" {
		b.WriteString(`<li id="wp-admin-bar-edit"><a class="ab-item" href="`)
		b.WriteString(html.EscapeString(editURL))
		b.WriteString(`">Edit</a></li>`)
	}
	b.WriteString(`</ul>`)
	b.WriteString(`<ul id="wp-admin-bar-top-secondary" class="ab-top-secondary ab-top-menu">`)
	b.WriteString(`<li id="wp-admin-bar-my-account" class="menupop with-avatar"><a class="ab-item" href="/wp-admin/profile.php">Howdy, `)
	b.WriteString(html.EscapeString(login))
	b.WriteString(`</a></li>`)
	// P7.128: front search form (wp_admin_bar_search_menu).
	b.WriteString(`<li id="wp-admin-bar-search" class="admin-bar-search"><div class="ab-item ab-empty-item" tabindex="-1">`)
	b.WriteString(`<form action="`)
	b.WriteString(html.EscapeString(home + "/"))
	b.WriteString(`" method="get" id="adminbarsearch">`)
	b.WriteString(`<input class="adminbar-input" name="s" id="adminbar-search" type="text" value="" maxlength="150" />`)
	b.WriteString(`<label for="adminbar-search" class="screen-reader-text">Search</label>`)
	b.WriteString(`<input type="submit" class="adminbar-button" value="Search" />`)
	b.WriteString(`</form></div></li>`)
	b.WriteString(`</ul></div></div>`)
	return b.String()
}

func adminBarEditURL(c *Context) string {
	if c == nil || c.Scope == nil || !c.Scope.IsSingular {
		return ""
	}
	id := c.Scope.QueriedObjectID
	if id <= 0 {
		return ""
	}
	if !c.CurrentUserCan("edit_post", id) {
		return ""
	}
	return fmt.Sprintf("/wp-admin/post.php?post=%s&action=edit", strconv.FormatInt(id, 10))
}
