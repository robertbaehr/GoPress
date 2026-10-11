package admin

import (
	"html"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// CoreUpdate is a subset of the update_core preferred response (P7.139).
// Response "upgrade" with Current set shows the admin update nag.
type CoreUpdate struct {
	Response string
	Current  string
}

// PreferredCoreUpdate mirrors get_preferred_from_update_core (P7.139).
// Core updates are out of scope for GoPress, so the default is nil (no nag).
// Filter pre_site_transient_update_core may return *CoreUpdate for tests.
func PreferredCoreUpdate(c *wp.Context) *CoreUpdate {
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("pre_site_transient_update_core", (*CoreUpdate)(nil))
		if u, ok := filtered.(*CoreUpdate); ok {
			return u
		}
	}
	return nil
}

// UpdateNag is update_nag (P7.139). Skips update-core.php; emits a warning
// notice with classes update-nag inline when an upgrade response is present.
func UpdateNag(c *wp.Context) {
	if c == nil {
		return
	}
	if adminPagenow(c) == "update-core.php" {
		return
	}
	cur := PreferredCoreUpdate(c)
	if cur == nil || cur.Response != "upgrade" {
		return
	}
	ver := strings.TrimSpace(cur.Current)
	if ver == "" {
		return
	}
	versionURL := "https://wordpress.org/documentation/wordpress-version/version-" +
		wp.SanitizeTitle(ver) + "/"
	var msg string
	if c.CurrentUserCan("update_core") {
		msg = `<a href="` + html.EscapeString(versionURL) + `">WordPress ` + html.EscapeString(ver) +
			`</a> is available! <a href="` + html.EscapeString(wp.AdminURL(c, "update-core.php")) +
			`" aria-label="Please update WordPress now">Please update now</a>.`
	} else {
		msg = `<a href="` + html.EscapeString(versionURL) + `">WordPress ` + html.EscapeString(ver) +
			`</a> is available! Please notify the site administrator.`
	}
	skip := false
	wp.EmitAdminNotice(c, msg, wp.AdminNoticeArgs{
		Type:              "warning",
		AdditionalClasses: []string{"update-nag", "inline"},
		ParagraphWrap:     &skip,
	})
}

func adminPagenow(c *wp.Context) string {
	if c == nil || c.Req == nil || c.Req.URL == nil {
		return ""
	}
	path := c.Req.URL.Path
	if i := strings.LastIndex(path, "/"); i >= 0 && i+1 < len(path) {
		return path[i+1:]
	}
	return path
}
