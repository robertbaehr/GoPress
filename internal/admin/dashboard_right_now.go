package admin

import (
	"fmt"
	"html"
	"path/filepath"
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// dashboardRightNowHTML builds the At a Glance widget body (P7.110 / wp_dashboard_right_now).
// Includes published post/page counts with post_status links, comment + moderation
// counts, dashboard_glance_items filter extras, and search-engines-discouraged notice.
func dashboardRightNowHTML(c *wp.Context) string {
	posts := countPostsByTypeStatus(c, "post", "publish")
	pages := countPostsByTypeStatus(c, "page", "publish")
	approved := countComments(c, "1")
	moderated := countComments(c, "0")

	var b strings.Builder
	b.WriteString(`<div id="dashboard_right_now" class="postbox">`)
	b.WriteString(`<h2 class="hndle"><span>At a Glance</span></h2>`)
	b.WriteString(`<div class="inside"><div class="main"><ul>`)

	if posts > 0 {
		label := "Published post"
		if posts != 1 {
			label = "Published posts"
		}
		b.WriteString(`<li class="post-count"><a href="/wp-admin/edit.php?post_status=publish&amp;post_type=post">`)
		b.WriteString(html.EscapeString(fmt.Sprintf("%d %s", posts, label)))
		b.WriteString(`</a></li>`)
	}
	if pages > 0 {
		label := "Published page"
		if pages != 1 {
			label = "Published pages"
		}
		b.WriteString(`<li class="page-count"><a href="/wp-admin/edit.php?post_status=publish&amp;post_type=page">`)
		b.WriteString(html.EscapeString(fmt.Sprintf("%d %s", pages, label)))
		b.WriteString(`</a></li>`)
	}
	if approved > 0 || moderated > 0 {
		label := "Comment"
		if approved != 1 {
			label = "Comments"
		}
		b.WriteString(`<li class="comment-count"><a href="/wp-admin/edit-comments.php">`)
		b.WriteString(html.EscapeString(fmt.Sprintf("%d %s", approved, label)))
		b.WriteString(`</a></li>`)
		modClass := "comment-mod-count"
		if moderated == 0 {
			modClass += " hidden"
		}
		modLabel := "Comment in moderation"
		if moderated != 1 {
			modLabel = "Comments in moderation"
		}
		b.WriteString(`<li class="`)
		b.WriteString(modClass)
		b.WriteString(`"><a href="/wp-admin/edit-comments.php?comment_status=moderated" class="comments-in-moderation-text">`)
		b.WriteString(html.EscapeString(fmt.Sprintf("%d %s", moderated, modLabel)))
		b.WriteString(`</a></li>`)
	}

	for _, item := range dashboardGlanceItems(c) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		b.WriteString("<li>")
		b.WriteString(item)
		b.WriteString("</li>\n")
	}

	b.WriteString(`</ul>`)
	// P7.132: update_right_now_message — WordPress version + active theme.
	b.WriteString(updateRightNowMessage(c))
	if blogPublicDiscouraged(c) {
		title := ""
		content := "Search engines discouraged"
		if c != nil && c.Hooks != nil {
			if v := c.Hooks.ApplyFilters("privacy_on_link_title", title); v != nil {
				if s, ok := v.(string); ok {
					title = s
				}
			}
			if v := c.Hooks.ApplyFilters("privacy_on_link_text", content); v != nil {
				if s, ok := v.(string); ok {
					content = s
				}
			}
		}
		titleAttr := ""
		if title != "" {
			titleAttr = ` title="` + html.EscapeString(title) + `"`
		}
		b.WriteString(`<p class="search-engines-info"><a href="/wp-admin/options-reading.php"`)
		b.WriteString(titleAttr)
		b.WriteString(`>`)
		b.WriteString(html.EscapeString(content))
		b.WriteString(`</a></p>`)
	}
	b.WriteString(`</div>`)
	// P7.132: rightnow_end / activity_box_end → optional .sub box.
	if sub := dashboardRightNowEndHTML(c); sub != "" {
		b.WriteString(`<div class="sub">`)
		b.WriteString(sub)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div></div>`)
	return b.String()
}

// updateRightNowMessage mirrors update_right_now_message() (P7.132).
func updateRightNowMessage(c *wp.Context) string {
	themeName := activeThemeDisplayName(c)
	themeHTML := html.EscapeString(themeName)
	if c != nil && c.CurrentUser != nil && c.UserCan(c.CurrentUser, "switch_themes") {
		themeHTML = `<a href="themes.php">` + themeHTML + `</a>`
	}
	content := "WordPress %1$s running %2$s theme."
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("update_right_now_text", content); v != nil {
			if s, ok := v.(string); ok && s != "" {
				content = s
			}
		}
	}
	msg := strings.Replace(content, "%1$s", html.EscapeString(wp.Version), 1)
	msg = strings.Replace(msg, "%2$s", themeHTML, 1)
	return `<p id="wp-version-message"><span id="wp-version">` + msg + `</span></p>`
}

func activeThemeDisplayName(c *wp.Context) string {
	stylesheet := wp.ActiveStylesheet(c)
	if stylesheet == "" {
		stylesheet = "twentytwentyfive"
	}
	abs := wp.AbsPath(c)
	if abs == "" {
		abs = "/workspace/"
	}
	styleCSS := filepath.Join(theme.ThemesDir(abs), stylesheet, "style.css")
	if data, err := theme.GetThemeData(styleCSS); err == nil && strings.TrimSpace(data.Name) != "" {
		return data.Name
	}
	if c != nil && c.Options != nil {
		if name := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("current_theme", ""))); name != "" {
			return name
		}
	}
	return stylesheet
}

// dashboardRightNowEndHTML runs rightnow_end + activity_box_end and captures
// WriteNotice output into the optional .sub box (P7.132).
func dashboardRightNowEndHTML(c *wp.Context) string {
	if c == nil || c.Hooks == nil {
		return ""
	}
	var buf strings.Builder
	c.Hooks.BeginNoticeCapture(&buf)
	c.Hooks.DoAction("rightnow_end")
	c.Hooks.DoAction("activity_box_end")
	c.Hooks.EndNoticeCapture()
	return strings.TrimSpace(buf.String())
}

func blogPublicDiscouraged(c *wp.Context) bool {
	if c == nil || c.Options == nil {
		return false
	}
	// Only show when the current user can manage options (WP right_now check).
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "manage_options") {
		return false
	}
	v := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("blog_public", "1")))
	return v == "0"
}

// dashboardGlanceItems runs the dashboard_glance_items filter (P7.110).
func dashboardGlanceItems(c *wp.Context) []string {
	items := []string{}
	if c == nil || c.Hooks == nil {
		return items
	}
	filtered := c.Hooks.ApplyFilters("dashboard_glance_items", items)
	switch t := filtered.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, v := range t {
			out = append(out, fmt.Sprint(v))
		}
		return out
	default:
		return items
	}
}

// activityRecentlyPublishedHTML lists recent published posts (P7.110).
func activityRecentlyPublishedHTML(c *wp.Context, limit int) string {
	if limit <= 0 {
		limit = 5
	}
	posts, err := recentPublishedPosts(c, limit)
	if err != nil || len(posts) == 0 {
		return `<div id="published-posts" class="activity-block"><h3>Recently Published</h3>
<p>Recent posts appear here when available.</p>
</div>`
	}
	var b strings.Builder
	b.WriteString(`<div id="published-posts" class="activity-block"><h3>Recently Published</h3><ul>`)
	for _, p := range posts {
		b.WriteString(`<li><span>`)
		b.WriteString(html.EscapeString(p.Date))
		b.WriteString(`</span> <a href="/wp-admin/post.php?post=`)
		b.WriteString(fmt.Sprintf("%d", p.ID))
		b.WriteString(`&amp;action=edit">`)
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString(`</a></li>`)
	}
	b.WriteString(`</ul></div>`)
	return b.String()
}

type recentDashPost struct {
	ID    int64
	Title string
	Date  string
}

func recentPublishedPosts(c *wp.Context, limit int) ([]recentDashPost, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_title, post_date FROM "+c.DB.Posts()+
			" WHERE post_type = %s AND post_status = %s ORDER BY post_date DESC LIMIT %d",
		"post", "publish", limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]recentDashPost, 0, len(rows))
	for _, row := range rows {
		title := strings.TrimSpace(fmt.Sprint(row["post_title"]))
		if title == "" {
			title = "(no title)"
		}
		out = append(out, recentDashPost{
			ID:    asInt64(row["ID"]),
			Title: title,
			Date:  strings.TrimSpace(fmt.Sprint(row["post_date"])),
		})
	}
	return out, nil
}
