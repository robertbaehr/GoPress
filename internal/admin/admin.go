// Package admin implements minimal wp-admin screen stubs (P7.1+).
package admin

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// Handler serves wp-admin screen stubs that require a logged-in user.
type Handler struct{}

// Handle dispatches known admin screen paths. Returns false when unhandled.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, c *wp.Context, path string) bool {
	switch path {
	case "/wp-admin", "/wp-admin/", "/wp-admin/index.php":
		h.requireLogin(w, r, c, h.serveDashboard)
		return true
	case "/wp-admin/edit.php":
		h.requireLogin(w, r, c, h.serveEditPosts)
		return true
	case "/wp-admin/edit-tags.php":
		h.requireLogin(w, r, c, h.serveEditTags)
		return true
	case "/wp-admin/post.php":
		h.requireLogin(w, r, c, h.servePostEdit)
		return true
	case "/wp-admin/options-general.php":
		h.requireLogin(w, r, c, h.serveOptionsGeneral)
		return true
	case "/wp-admin/options-writing.php":
		h.requireLogin(w, r, c, h.serveOptionsWriting)
		return true
	case "/wp-admin/options-reading.php":
		h.requireLogin(w, r, c, h.serveOptionsReading)
		return true
	case "/wp-admin/options-discussion.php":
		h.requireLogin(w, r, c, h.serveOptionsDiscussion)
		return true
	case "/wp-admin/options-media.php":
		h.requireLogin(w, r, c, h.serveOptionsMedia)
		return true
	case "/wp-admin/options-permalink.php":
		h.requireLogin(w, r, c, h.serveOptionsPermalink)
		return true
	case "/wp-admin/plugins.php":
		h.requireLogin(w, r, c, h.servePlugins)
		return true
	case "/wp-admin/profile.php":
		h.requireLogin(w, r, c, h.serveProfile)
		return true
	case "/wp-admin/upload.php":
		h.requireLogin(w, r, c, h.serveUpload)
		return true
	case "/wp-admin/media-new.php":
		h.requireLogin(w, r, c, h.serveMediaNew)
		return true
	case "/wp-admin/edit-comments.php":
		h.requireLogin(w, r, c, h.serveEditComments)
		return true
	case "/wp-admin/comment.php":
		h.requireLogin(w, r, c, h.serveCommentModerate)
		return true
	case "/wp-admin/themes.php":
		h.requireLogin(w, r, c, h.serveThemes)
		return true
	case "/wp-admin/users.php":
		h.requireLogin(w, r, c, h.serveUsers)
		return true
	case "/wp-admin/user-new.php":
		h.requireLogin(w, r, c, h.serveUserNew)
		return true
	case "/wp-admin/site-editor.php":
		h.requireLogin(w, r, c, h.serveSiteEditor)
		return true
	case "/wp-admin/nav-menus.php":
		h.requireLogin(w, r, c, h.serveNavMenus)
		return true
	case "/wp-admin/customize.php":
		h.requireLogin(w, r, c, h.serveCustomizerGone)
		return true
	case "/wp-admin/theme-install.php":
		h.requireLogin(w, r, c, h.serveThemeInstallGone)
		return true
	case "/wp-admin/theme-editor.php":
		h.requireLogin(w, r, c, h.serveThemeEditorGone)
		return true
	case "/wp-admin/plugin-install.php":
		h.requireLogin(w, r, c, h.servePluginInstallGone)
		return true
	case "/wp-admin/plugin-editor.php":
		h.requireLogin(w, r, c, h.servePluginEditorGone)
		return true
	case "/wp-admin/update-core.php":
		h.requireLogin(w, r, c, h.serveUpdateCoreGone)
		return true
	case "/wp-admin/link-manager.php", "/wp-admin/link-add.php", "/wp-admin/link.php":
		h.requireLogin(w, r, c, h.serveLinkManagerGone)
		return true
	case "/wp-admin/admin-ajax.php":
		h.serveAdminAjax(w, r, c)
		return true
	case "/wp-admin/async-upload.php":
		h.requireLogin(w, r, c, h.serveAsyncUpload)
		return true
	case "/wp-admin/export-personal-data.php":
		h.requireLogin(w, r, c, h.serveExportPersonalData)
		return true
	case "/wp-admin/erase-personal-data.php":
		h.requireLogin(w, r, c, h.serveErasePersonalData)
		return true
	case "/wp-admin/options-privacy.php":
		h.requireLogin(w, r, c, h.serveOptionsPrivacy)
		return true
	case "/wp-admin/privacy-policy-guide.php":
		h.requireLogin(w, r, c, h.servePrivacyPolicyGuide)
		return true
	case "/wp-admin/revision.php":
		h.requireLogin(w, r, c, h.serveRevisionCompare)
		return true
	case "/wp-admin/export.php":
		h.requireLogin(w, r, c, h.serveExport)
		return true
	case "/wp-admin/import.php":
		h.requireLogin(w, r, c, h.serveImport)
		return true
	case "/wp-admin/tools.php":
		h.requireLogin(w, r, c, h.serveTools)
		return true
	case "/wp-admin/site-health.php":
		h.requireLogin(w, r, c, h.serveSiteHealth)
		return true
	default:
		return false
	}
}

type screenFunc func(w http.ResponseWriter, r *http.Request, c *wp.Context)

func (h *Handler) requireLogin(w http.ResponseWriter, r *http.Request, c *wp.Context, next screenFunc) {
	if c == nil {
		http.Error(w, "GoPress context unavailable\n", http.StatusServiceUnavailable)
		return
	}
	c.AuthenticateRequest()
	if c.CurrentUser == nil {
		redirect := r.URL.RequestURI()
		if redirect == "" {
			redirect = "/wp-admin/"
		}
		target := "/wp-login.php?redirect_to=" + url.QueryEscape(redirect)
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	next(w, r, c)
}

func (h *Handler) serveDashboard(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	restNonce := c.CreateNonce("wp_rest")
	hbNonce := c.CreateNonce("heartbeat-nonce")
	posts := countPostsByTypeStatus(c, "post", "publish")
	pages := countPostsByTypeStatus(c, "page", "publish")
	comments := countComments(c, "1")
	ch := newAdminChrome(c, "Dashboard", "index-php")
	ch.HeadExtra += fmt.Sprintf(
		`<script>var wpApiSettings = {"root":"/wp-json/","nonce":"%s"};</script>
<script>var heartbeatSettings = {"nonce":"%s"};</script>
`, html.EscapeString(restNonce), html.EscapeString(hbNonce))
	// Secondary dashboard widgets (P7.40): Activity, Quick Draft, Events and News
	// (primary + secondary / "Other WordPress News" stub — no remote RSS fetch).
	inner := fmt.Sprintf(`%s<h1>Dashboard</h1>
<p>%s</p>
<div id="dashboard-widgets-wrap">
<div id="dashboard-widgets" class="metabox-holder">
<div id="postbox-container-1" class="postbox-container">
<div id="dashboard_right_now" class="postbox">
<h2 class="hndle"><span>At a Glance</span></h2>
<div class="inside">
<ul>
<li class="post-count"><a href="/wp-admin/edit.php">%d Posts</a></li>
<li class="page-count"><a href="/wp-admin/edit.php?post_type=page">%d Pages</a></li>
<li class="comment-count"><a href="/wp-admin/edit-comments.php">%d Comments</a></li>
</ul>
</div>
</div>
<div id="dashboard_activity" class="postbox">
<h2 class="hndle"><span>Activity</span></h2>
<div class="inside">
<div id="activity-widget">
<div id="published-posts" class="activity-block"><h3>Recently Published</h3>
<p>Recent posts appear here when available.</p>
</div>
<div id="latest-comments" class="activity-block"><h3>Recent Comments</h3>
<p>No comments yet. <a href="/wp-admin/edit-comments.php">View all</a></p>
</div>
</div>
</div>
</div>
</div>
<div id="postbox-container-2" class="postbox-container">
<div id="dashboard_quick_press" class="postbox">
<h2 class="hndle"><span>Quick Draft</span></h2>
<div class="inside">
<form name="post" action="/wp-admin/post.php" method="post" id="quick-press">
<div class="input-text-wrap"><label for="title">Title</label>
<input type="text" name="post_title" id="title" autocomplete="off" /></div>
<div class="textarea-wrap"><label for="content">Content</label>
<textarea name="content" id="content" rows="3" cols="15" autocomplete="off"></textarea></div>
<p class="submit"><input type="submit" name="save" id="save-post" class="button button-primary" value="Save Draft" />
<br class="clear" /></p>
</form>
</div>
</div>
<div id="dashboard_primary" class="postbox">
<h2 class="hndle"><span>WordPress Events and News</span></h2>
<div class="inside">
<div class="rss-widget">
<ul>
<li><a href="https://wordpress.org/news/">WordPress Blog</a>
<span class="rss-date">News feed stub (remote RSS not fetched)</span></li>
</ul>
</div>
<div class="rss-widget" id="dashboard_secondary">
<h3>Other WordPress News</h3>
<ul>
<li><a class="rsswidget" href="https://planet.wordpress.org/">WordPress Planet</a>
<span class="rss-date">Secondary news stub (dashboard_secondary)</span></li>
</ul>
</div>
</div>
</div>
</div>
</div>
</div>
`, adminNoticesHTML(c), html.EscapeString(blogname), posts, pages, comments)
	writeAdminPage(w, c, ch, inner)
}

func (h *Handler) serveEditPosts(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	postType := r.URL.Query().Get("post_type")
	if postType == "" {
		postType = "post"
	}
	if postType != "post" && postType != "page" {
		postType = "post"
	}
	posts, err := recentPostsByType(c, postType, 20)
	if err != nil {
		http.Error(w, "posts: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	heading := "Posts"
	if postType == "page" {
		heading = "Pages"
	}
	var b strings.Builder
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(heading))
	b.WriteString("</h1>\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped table-view-list posts\">\n")
	b.WriteString("<thead><tr>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-title column-primary\">Title</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-author\">Author</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-date\">Date</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-status\">Status</th>")
	b.WriteString("</tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	for _, p := range posts {
		b.WriteString("<tr id=\"post-")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\" class=\"iedit author-self status-")
		b.WriteString(html.EscapeString(p.Status))
		b.WriteString(" type-")
		b.WriteString(html.EscapeString(postType))
		b.WriteString(" hentry\">")
		b.WriteString("<td class=\"title column-title has-row-actions column-primary page-title\" data-colname=\"Title\"><strong>")
		b.WriteString("<a class=\"row-title\" href=\"/wp-admin/post.php?post=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("&amp;action=edit\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></strong>")
		b.WriteString("<div class=\"row-actions\"><span class=\"edit\"><a href=\"/wp-admin/post.php?post=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("&amp;action=edit\">Edit</a></span></div></td>")
		b.WriteString("<td class=\"author column-author\" data-colname=\"Author\">")
		b.WriteString(html.EscapeString(p.Author))
		b.WriteString("</td>")
		b.WriteString("<td class=\"date column-date\" data-colname=\"Date\"><abbr>")
		b.WriteString(html.EscapeString(displayPostDateLabel(p.Status)))
		b.WriteString("</abbr><br>")
		b.WriteString(html.EscapeString(strings.TrimSpace(p.Date)))
		b.WriteString("</td>")
		b.WriteString("<td class=\"status column-status\" data-colname=\"Status\">")
		b.WriteString(html.EscapeString(displayPostStatus(p.Status)))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	_ = blogname
	writeAdminPage(w, c, newAdminChrome(c, heading, "edit-php"), b.String())
}

// serveEditTags is the Categories / Tags list stub (P7.29 / edit-tags.php).
func (h *Handler) serveEditTags(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	taxonomy := strings.TrimSpace(r.URL.Query().Get("taxonomy"))
	if taxonomy == "" {
		taxonomy = "category"
	}
	heading := "Categories"
	screen := "edit-tags-php"
	switch taxonomy {
	case "post_tag", "tag":
		taxonomy = "post_tag"
		heading = "Tags"
	case "category":
		// ok
	default:
		taxonomy = "category"
	}
	terms, err := listAdminTerms(c, taxonomy, 50)
	if err != nil {
		http.Error(w, "terms: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<h1 class=\"wp-heading-inline\">")
	b.WriteString(html.EscapeString(heading))
	b.WriteString("</h1>\n")
	b.WriteString("<hr class=\"wp-header-end\">\n")
	b.WriteString("<p class=\"description\">GoPress edit-tags stub — create/edit via REST for now.</p>\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped table-view-list tags\">\n")
	b.WriteString("<thead><tr>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-name column-primary\">Name</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-description\">Description</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-slug\">Slug</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-posts\">Count</th>")
	b.WriteString("</tr></thead>\n<tbody id=\"the-list\">\n")
	if len(terms) == 0 {
		b.WriteString("<tr class=\"no-items\"><td class=\"colspanchange\" colspan=\"4\">No terms found.</td></tr>\n")
	}
	for _, t := range terms {
		b.WriteString("<tr id=\"tag-")
		b.WriteString(strconv.FormatInt(t.ID, 10))
		b.WriteString("\">")
		b.WriteString("<td class=\"name column-name has-row-actions column-primary\" data-colname=\"Name\"><strong>")
		b.WriteString(html.EscapeString(t.Name))
		b.WriteString("</strong></td>")
		b.WriteString("<td class=\"description column-description\" data-colname=\"Description\">")
		b.WriteString(html.EscapeString(t.Description))
		b.WriteString("</td>")
		b.WriteString("<td class=\"slug column-slug\" data-colname=\"Slug\">")
		b.WriteString(html.EscapeString(t.Slug))
		b.WriteString("</td>")
		b.WriteString("<td class=\"posts column-posts\" data-colname=\"Count\">")
		b.WriteString(strconv.FormatInt(t.Count, 10))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	writeAdminPage(w, c, newAdminChrome(c, heading, screen), b.String())
}

type adminTerm struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	Count       int64
}

func listAdminTerms(c *wp.Context, taxonomy string, limit int) ([]adminTerm, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 50
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.term_id, t.name, t.slug, tt.description, tt.count FROM "+
			c.DB.Terms()+" t INNER JOIN "+c.DB.TermTaxonomy()+
			" tt ON t.term_id = tt.term_id WHERE tt.taxonomy = %s ORDER BY t.name ASC LIMIT %d",
		taxonomy, limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]adminTerm, 0, len(rows))
	for _, row := range rows {
		out = append(out, adminTerm{
			ID:          asInt64(row["term_id"]),
			Name:        asString(row["name"]),
			Slug:        asString(row["slug"]),
			Description: asString(row["description"]),
			Count:       asInt64(row["count"]),
		})
	}
	return out, nil
}

func displayPostStatus(status string) string {
	switch status {
	case "publish":
		return "Published"
	case "draft":
		return "Draft"
	case "pending":
		return "Pending Review"
	case "private":
		return "Private"
	case "future":
		return "Scheduled"
	case "trash":
		return "Trash"
	case "":
		return "—"
	default:
		if len(status) == 1 {
			return strings.ToUpper(status)
		}
		return strings.ToUpper(status[:1]) + status[1:]
	}
}

func displayPostDateLabel(status string) string {
	switch status {
	case "draft", "pending", "auto-draft":
		return "Last Modified"
	case "future":
		return "Scheduled"
	default:
		return "Published"
	}
}

func (h *Handler) serveOptionsGeneral(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<h1>General Settings</h1>\n")
	b.WriteString("<form method=\"post\" action=\"options.php\">\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr>\n<th scope=\"row\"><label for=\"blogname\">Site Title</label></th>\n")
	b.WriteString("<td><input name=\"blogname\" type=\"text\" id=\"blogname\" value=\"")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("\" class=\"regular-text\"></td>\n</tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Save Changes\"></p>\n")
	b.WriteString("</form>\n")
	writeAdminPage(w, c, newAdminChrome(c, "General Settings", "options-general-php"), b.String())
}

// serveOptionsWriting is the Settings → Writing screen stub (P7.13).
func (h *Handler) serveOptionsWriting(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	defaultCat := optionString(c, "default_category", "1")
	defaultFormat := optionString(c, "default_post_format", "0")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Writing Settings &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Writing Settings</h1>\n")
	b.WriteString("<p class=\"description\">GoPress writing settings stub — save lands with options.php.</p>\n")
	b.WriteString("<form method=\"post\" action=\"options.php\">\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr>\n<th scope=\"row\"><label for=\"default_category\">Default Post Category</label></th>\n")
	b.WriteString("<td><input name=\"default_category\" type=\"text\" id=\"default_category\" value=\"")
	b.WriteString(html.EscapeString(defaultCat))
	b.WriteString("\" class=\"small-text\"></td>\n</tr>\n")
	b.WriteString("<tr>\n<th scope=\"row\"><label for=\"default_post_format\">Default Post Format</label></th>\n")
	b.WriteString("<td><select name=\"default_post_format\" id=\"default_post_format\">\n")
	b.WriteString("<option value=\"0\"")
	if defaultFormat == "" || defaultFormat == "0" {
		b.WriteString(" selected=\"selected\"")
	}
	b.WriteString(">Standard</option>\n")
	b.WriteString("</select></td>\n</tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Save Changes\"></p>\n")
	b.WriteString("</form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveOptionsReading is the Settings → Reading screen stub (P7.14).
func (h *Handler) serveOptionsReading(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	showOnFront := optionString(c, "show_on_front", "posts")
	postsPerPage := optionString(c, "posts_per_page", "10")
	blogPublic := optionString(c, "blog_public", "1")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Reading Settings &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Reading Settings</h1>\n")
	b.WriteString("<p class=\"description\">GoPress reading settings stub — save lands with options.php.</p>\n")
	b.WriteString("<form method=\"post\" action=\"options.php\">\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr>\n<th scope=\"row\">Your homepage displays</th>\n<td><fieldset>\n")
	b.WriteString("<label><input name=\"show_on_front\" type=\"radio\" value=\"posts\"")
	if showOnFront != "page" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Your latest posts</label><br>\n")
	b.WriteString("<label><input name=\"show_on_front\" type=\"radio\" value=\"page\"")
	if showOnFront == "page" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> A static page</label>\n")
	b.WriteString("</fieldset></td>\n</tr>\n")
	b.WriteString("<tr>\n<th scope=\"row\"><label for=\"posts_per_page\">Blog pages show at most</label></th>\n")
	b.WriteString("<td><input name=\"posts_per_page\" type=\"number\" step=\"1\" min=\"1\" id=\"posts_per_page\" value=\"")
	b.WriteString(html.EscapeString(postsPerPage))
	b.WriteString("\" class=\"small-text\"> posts</td>\n</tr>\n")
	b.WriteString("<tr>\n<th scope=\"row\">Search engine visibility</th>\n<td>\n")
	b.WriteString("<label for=\"blog_public\"><input name=\"blog_public\" type=\"checkbox\" id=\"blog_public\" value=\"0\"")
	if blogPublic == "0" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Discourage search engines from indexing this site</label>\n")
	b.WriteString("</td>\n</tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Save Changes\"></p>\n")
	b.WriteString("</form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveOptionsDiscussion is the Settings → Discussion screen stub (P7.22).
func (h *Handler) serveOptionsDiscussion(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	defaultCommentStatus := optionString(c, "default_comment_status", "open")
	defaultPingStatus := optionString(c, "default_ping_status", "open")
	requireNameEmail := optionString(c, "require_name_email", "1")
	commentRegistration := optionString(c, "comment_registration", "0")
	closeOld := optionString(c, "close_comments_for_old_posts", "0")
	closeDays := optionString(c, "close_comments_days_old", "14")
	threadComments := optionString(c, "thread_comments", "1")
	threadDepth := optionString(c, "thread_comments_depth", "5")
	pageComments := optionString(c, "page_comments", "0")
	commentsPerPage := optionString(c, "comments_per_page", "50")
	commentsNotify := optionString(c, "comments_notify", "1")
	moderationNotify := optionString(c, "moderation_notify", "1")
	commentModeration := optionString(c, "comment_moderation", "0")
	previouslyApproved := optionString(c, "comment_previously_approved", "1")
	showAvatars := optionString(c, "show_avatars", "1")

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Discussion Settings &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Discussion Settings</h1>\n")
	b.WriteString("<p class=\"description\">GoPress discussion settings stub — save lands with options.php.</p>\n")
	b.WriteString("<form method=\"post\" action=\"options.php\">\n")
	b.WriteString("<table class=\"form-table indent-children\" role=\"presentation\">\n")

	b.WriteString("<tr>\n<th scope=\"row\">Default post settings</th>\n<td><fieldset>\n")
	b.WriteString("<label for=\"default_ping_status\"><input name=\"default_ping_status\" type=\"checkbox\" id=\"default_ping_status\" value=\"open\"")
	if defaultPingStatus == "open" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Allow link notifications from other blogs (pingbacks and trackbacks) on new posts</label><br>\n")
	b.WriteString("<label for=\"default_comment_status\"><input name=\"default_comment_status\" type=\"checkbox\" id=\"default_comment_status\" value=\"open\"")
	if defaultCommentStatus == "open" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Allow people to submit comments on new posts</label>\n")
	b.WriteString("</fieldset></td>\n</tr>\n")

	b.WriteString("<tr>\n<th scope=\"row\">Other comment settings</th>\n<td><fieldset>\n")
	b.WriteString("<label for=\"require_name_email\"><input type=\"checkbox\" name=\"require_name_email\" id=\"require_name_email\" value=\"1\"")
	if requireNameEmail == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Comment author must fill out name and email</label><br>\n")
	b.WriteString("<label for=\"comment_registration\"><input name=\"comment_registration\" type=\"checkbox\" id=\"comment_registration\" value=\"1\"")
	if commentRegistration == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Users must be registered and logged in to comment</label><br>\n")
	b.WriteString("<input name=\"close_comments_for_old_posts\" type=\"checkbox\" id=\"close_comments_for_old_posts\" value=\"1\"")
	if closeOld == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> <label for=\"close_comments_for_old_posts\">Automatically close comments on old posts</label>\n")
	b.WriteString("<ul><li><label for=\"close_comments_days_old\">Close comments when post is how many days old</label> ")
	b.WriteString("<input name=\"close_comments_days_old\" type=\"number\" step=\"1\" min=\"0\" id=\"close_comments_days_old\" value=\"")
	b.WriteString(html.EscapeString(closeDays))
	b.WriteString("\" class=\"small-text\" /></li></ul>\n")
	b.WriteString("<input name=\"thread_comments\" type=\"checkbox\" id=\"thread_comments\" value=\"1\"")
	if threadComments == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> <label for=\"thread_comments\">Enable threaded (nested) comments</label>\n")
	b.WriteString("<ul><li><label for=\"thread_comments_depth\">Number of levels for threaded (nested) comments</label> ")
	b.WriteString("<select name=\"thread_comments_depth\" id=\"thread_comments_depth\">\n")
	for _, d := range []string{"2", "3", "4", "5", "6", "7", "8", "9", "10"} {
		b.WriteString("<option value=\"")
		b.WriteString(d)
		b.WriteString("\"")
		if threadDepth == d {
			b.WriteString(" selected=\"selected\"")
		}
		b.WriteString(">")
		b.WriteString(d)
		b.WriteString("</option>\n")
	}
	b.WriteString("</select></li></ul>\n")
	b.WriteString("<input name=\"page_comments\" type=\"checkbox\" id=\"page_comments\" value=\"1\"")
	if pageComments == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> <label for=\"page_comments\">Break comments into pages</label>\n")
	b.WriteString("<ul><li><label for=\"comments_per_page\">Top level comments per page</label> ")
	b.WriteString("<input name=\"comments_per_page\" type=\"number\" step=\"1\" min=\"0\" id=\"comments_per_page\" value=\"")
	b.WriteString(html.EscapeString(commentsPerPage))
	b.WriteString("\" class=\"small-text\" /></li></ul>\n")
	b.WriteString("</fieldset></td>\n</tr>\n")

	b.WriteString("<tr>\n<th scope=\"row\">Email me whenever</th>\n<td><fieldset>\n")
	b.WriteString("<label for=\"comments_notify\"><input name=\"comments_notify\" type=\"checkbox\" id=\"comments_notify\" value=\"1\"")
	if commentsNotify == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Anyone posts a comment</label><br>\n")
	b.WriteString("<label for=\"moderation_notify\"><input name=\"moderation_notify\" type=\"checkbox\" id=\"moderation_notify\" value=\"1\"")
	if moderationNotify == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> A comment is held for moderation</label>\n")
	b.WriteString("</fieldset></td>\n</tr>\n")

	b.WriteString("<tr>\n<th scope=\"row\">Before a comment appears</th>\n<td><fieldset>\n")
	b.WriteString("<label for=\"comment_moderation\"><input name=\"comment_moderation\" type=\"checkbox\" id=\"comment_moderation\" value=\"1\"")
	if commentModeration == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Comment must be manually approved</label><br>\n")
	b.WriteString("<label for=\"comment_previously_approved\"><input type=\"checkbox\" name=\"comment_previously_approved\" id=\"comment_previously_approved\" value=\"1\"")
	if previouslyApproved == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Comment author must have a previously approved comment</label>\n")
	b.WriteString("</fieldset></td>\n</tr>\n")

	b.WriteString("<tr>\n<th scope=\"row\">Avatars</th>\n<td>\n")
	b.WriteString("<label for=\"show_avatars\"><input name=\"show_avatars\" type=\"checkbox\" id=\"show_avatars\" value=\"1\"")
	if showAvatars == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString("> Show Avatars</label>\n")
	b.WriteString("</td>\n</tr>\n")

	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Save Changes\"></p>\n")
	b.WriteString("</form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveOptionsMedia is the Settings → Media screen stub (P7.23).
func (h *Handler) serveOptionsMedia(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	thumbW := optionString(c, "thumbnail_size_w", "150")
	thumbH := optionString(c, "thumbnail_size_h", "150")
	thumbCrop := optionString(c, "thumbnail_crop", "1")
	mediumW := optionString(c, "medium_size_w", "300")
	mediumH := optionString(c, "medium_size_h", "300")
	largeW := optionString(c, "large_size_w", "1024")
	largeH := optionString(c, "large_size_h", "1024")
	embeds := optionString(c, "embed_autourls", "1")

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Media Settings &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Media Settings</h1>\n")
	b.WriteString("<p class=\"description\">GoPress media settings stub — save lands with options.php.</p>\n")
	b.WriteString("<form action=\"options.php\" method=\"post\">\n")
	b.WriteString("<h2 id=\"wp-settings-section-image-sizes\" class=\"title\">Image sizes</h2>\n")
	b.WriteString("<p>The sizes listed below determine the maximum dimensions in pixels to use when adding an image to the Media Library.</p>\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")

	b.WriteString("<tr>\n<th scope=\"row\">Thumbnail size</th>\n<td><fieldset>\n")
	b.WriteString("<label for=\"thumbnail_size_w\">Width</label> ")
	b.WriteString("<input name=\"thumbnail_size_w\" type=\"number\" step=\"1\" min=\"0\" id=\"thumbnail_size_w\" value=\"")
	b.WriteString(html.EscapeString(thumbW))
	b.WriteString("\" class=\"small-text\" /><br>\n")
	b.WriteString("<label for=\"thumbnail_size_h\">Height</label> ")
	b.WriteString("<input name=\"thumbnail_size_h\" type=\"number\" step=\"1\" min=\"0\" id=\"thumbnail_size_h\" value=\"")
	b.WriteString(html.EscapeString(thumbH))
	b.WriteString("\" class=\"small-text\" />\n")
	b.WriteString("</fieldset>\n")
	b.WriteString("<input name=\"thumbnail_crop\" type=\"checkbox\" id=\"thumbnail_crop\" value=\"1\"")
	if thumbCrop == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString(" /> <label for=\"thumbnail_crop\">Crop thumbnail to exact dimensions (normally thumbnails are proportional)</label>\n")
	b.WriteString("</td>\n</tr>\n")

	b.WriteString("<tr>\n<th scope=\"row\">Medium size</th>\n<td><fieldset>\n")
	b.WriteString("<label for=\"medium_size_w\">Max Width</label> ")
	b.WriteString("<input name=\"medium_size_w\" type=\"number\" step=\"1\" min=\"0\" id=\"medium_size_w\" value=\"")
	b.WriteString(html.EscapeString(mediumW))
	b.WriteString("\" class=\"small-text\" /><br>\n")
	b.WriteString("<label for=\"medium_size_h\">Max Height</label> ")
	b.WriteString("<input name=\"medium_size_h\" type=\"number\" step=\"1\" min=\"0\" id=\"medium_size_h\" value=\"")
	b.WriteString(html.EscapeString(mediumH))
	b.WriteString("\" class=\"small-text\" />\n")
	b.WriteString("</fieldset></td>\n</tr>\n")

	b.WriteString("<tr>\n<th scope=\"row\">Large size</th>\n<td><fieldset>\n")
	b.WriteString("<label for=\"large_size_w\">Max Width</label> ")
	b.WriteString("<input name=\"large_size_w\" type=\"number\" step=\"1\" min=\"0\" id=\"large_size_w\" value=\"")
	b.WriteString(html.EscapeString(largeW))
	b.WriteString("\" class=\"small-text\" /><br>\n")
	b.WriteString("<label for=\"large_size_h\">Max Height</label> ")
	b.WriteString("<input name=\"large_size_h\" type=\"number\" step=\"1\" min=\"0\" id=\"large_size_h\" value=\"")
	b.WriteString(html.EscapeString(largeH))
	b.WriteString("\" class=\"small-text\" />\n")
	b.WriteString("</fieldset></td>\n</tr>\n")

	b.WriteString("</table>\n")
	b.WriteString("<h2 class=\"title\">Embeds</h2>\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr>\n<th scope=\"row\">Auto-embeds</th>\n<td>\n")
	b.WriteString("<label for=\"embed_autourls\"><input name=\"embed_autourls\" type=\"checkbox\" id=\"embed_autourls\" value=\"1\"")
	if embeds == "1" {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString(" /> Items linked to should be embedded automatically</label>\n")
	b.WriteString("</td>\n</tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Save Changes\"></p>\n")
	b.WriteString("</form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveOptionsPermalink is the Settings → Permalinks screen stub (P7.36).
// Shows common permalink structure radios plus optional category/tag bases.
func (h *Handler) serveOptionsPermalink(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	structure := optionString(c, "permalink_structure", "")
	categoryBase := optionString(c, "category_base", "")
	tagBase := optionString(c, "tag_base", "")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	if home == "" {
		home = "http://127.0.0.1:8082"
	}

	type structOpt struct {
		id, label, value, example string
	}
	opts := []structOpt{
		{"plain", "Plain", "", home + "/?p=123"},
		{"day-name", "Day and name", "/%year%/%monthnum%/%day%/%postname%/", home + "/2024/01/01/sample-post/"},
		{"month-name", "Month and name", "/%year%/%monthnum%/%postname%/", home + "/2024/01/sample-post/"},
		{"numeric", "Numeric", "/archives/%post_id%", home + "/archives/123"},
		{"post-name", "Post name", "/%postname%/", home + "/sample-post/"},
	}
	known := map[string]bool{}
	for _, o := range opts {
		known[o.value] = true
	}
	customChecked := structure != "" && !known[structure]

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Permalink Settings &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Permalink Settings</h1>\n")
	b.WriteString("<p class=\"description\">GoPress permalink settings stub — structure display only; save lands with options-permalink.php.</p>\n")
	b.WriteString("<form name=\"form\" action=\"options-permalink.php\" method=\"post\">\n")
	b.WriteString("<h2 id=\"wp-settings-section-common-settings\" class=\"title\">Common Settings</h2>\n")
	b.WriteString("<table class=\"form-table permalink-structure\" role=\"presentation\">\n")
	b.WriteString("<tr>\n<th scope=\"row\">Permalink structure</th>\n<td>\n")
	b.WriteString("<fieldset class=\"structure-selection\">\n")
	b.WriteString("<legend class=\"screen-reader-text\">Permalink structure</legend>\n")
	for _, o := range opts {
		b.WriteString("<div class=\"row\">\n")
		b.WriteString("<input id=\"permalink-input-")
		b.WriteString(html.EscapeString(o.id))
		b.WriteString("\" name=\"selection\" type=\"radio\" value=\"")
		b.WriteString(html.EscapeString(o.value))
		b.WriteString("\"")
		if structure == o.value && !customChecked {
			b.WriteString(" checked=\"checked\"")
		}
		b.WriteString(" />\n")
		b.WriteString("<div><label for=\"permalink-input-")
		b.WriteString(html.EscapeString(o.id))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(o.label))
		b.WriteString("</label>\n<p><code id=\"permalink-")
		b.WriteString(html.EscapeString(o.id))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(o.example))
		b.WriteString("</code></p></div>\n</div>\n")
	}
	b.WriteString("<div class=\"row\">\n")
	b.WriteString("<input id=\"custom_selection\" name=\"selection\" type=\"radio\" value=\"custom\"")
	if customChecked {
		b.WriteString(" checked=\"checked\"")
	}
	b.WriteString(" />\n")
	b.WriteString("<div><label for=\"custom_selection\">Custom Structure</label>\n")
	b.WriteString("<p><code>")
	b.WriteString(html.EscapeString(home))
	customVal := structure
	if !customChecked {
		customVal = ""
	}
	b.WriteString("</code> <input name=\"permalink_structure\" id=\"permalink_structure\" type=\"text\" value=\"")
	b.WriteString(html.EscapeString(customVal))
	b.WriteString("\" class=\"regular-text code\" /></p></div>\n</div>\n")
	b.WriteString("</fieldset>\n</td>\n</tr>\n</table>\n")

	b.WriteString("<h2 class=\"title\">Optional</h2>\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr>\n<th scope=\"row\"><label for=\"category_base\">Category base</label></th>\n")
	b.WriteString("<td><input name=\"category_base\" id=\"category_base\" type=\"text\" value=\"")
	b.WriteString(html.EscapeString(categoryBase))
	b.WriteString("\" class=\"regular-text code\" /></td>\n</tr>\n")
	b.WriteString("<tr>\n<th scope=\"row\"><label for=\"tag_base\">Tag base</label></th>\n")
	b.WriteString("<td><input name=\"tag_base\" id=\"tag_base\" type=\"text\" value=\"")
	b.WriteString(html.EscapeString(tagBase))
	b.WriteString("\" class=\"regular-text code\" /></td>\n</tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Save Changes\"></p>\n")
	b.WriteString("</form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) servePlugins(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	notice := ""
	if r.Method == http.MethodPost || r.URL.Query().Get("action") != "" {
		msg, err := handlePluginAction(r, c)
		if err != nil {
			http.Error(w, "plugins: "+err.Error()+"\n", http.StatusForbidden)
			return
		}
		if msg != "" {
			// PRG after mutation.
			http.Redirect(w, r, "/wp-admin/plugins.php?"+msg, http.StatusFound)
			return
		}
	}
	if r.URL.Query().Get("activate") == "true" {
		notice = "activate"
	}
	if r.URL.Query().Get("deactivate") == "true" {
		notice = "deactivate"
	}

	var b strings.Builder
	b.WriteString("<h1>Plugins</h1>\n")
	switch notice {
	case "activate":
		b.WriteString("<div id=\"message\" class=\"updated notice is-dismissible\"><p>Plugin <strong>activated</strong>.</p></div>\n")
	case "deactivate":
		b.WriteString("<div id=\"message\" class=\"updated notice is-dismissible\"><p>Plugin <strong>deactivated</strong>.</p></div>\n")
	}
	// P9.8: Akismet is not part of the GoPress distribution.
	b.WriteString(`<div class="notice notice-info" id="gopress-akismet-gone"><p>Akismet is <strong>not bundled</strong> with GoPress and is out of scope.</p></div>` + "\n")
	b.WriteString("<table class=\"wp-list-table widefat plugins\">\n")
	b.WriteString("<thead><tr><th scope=\"col\">Plugin</th><th scope=\"col\">Description</th></tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	installed, err := plugins.ListInstalledPlugins(c)
	if err != nil {
		http.Error(w, "plugins: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if len(installed) == 0 {
		b.WriteString("<tr class=\"inactive\"><td class=\"plugin-title\"><strong>No plugins installed</strong></td>")
		b.WriteString("<td class=\"column-description desc\"><p>GoPress plugin management stub (P7.1).</p></td></tr>\n")
	}
	for _, p := range installed {
		active := plugins.IsPluginActive(c, p.File)
		rowClass := "inactive"
		if active {
			rowClass = "active"
		}
		b.WriteString("<tr class=\"")
		b.WriteString(rowClass)
		b.WriteString("\" data-plugin=\"")
		b.WriteString(html.EscapeString(p.File))
		b.WriteString("\"><td class=\"plugin-title\"><strong>")
		b.WriteString(html.EscapeString(p.Name))
		b.WriteString("</strong>")
		b.WriteString("<div class=\"row-actions visible\">")
		if active {
			nonce := c.CreateNonce("deactivate-plugin_" + p.File)
			b.WriteString("<span class=\"deactivate\"><a href=\"/wp-admin/plugins.php?action=deactivate&amp;plugin=")
			b.WriteString(html.EscapeString(url.QueryEscape(p.File)))
			b.WriteString("&amp;_wpnonce=")
			b.WriteString(html.EscapeString(nonce))
			b.WriteString("\">Deactivate</a></span>")
		} else {
			nonce := c.CreateNonce("activate-plugin_" + p.File)
			b.WriteString("<span class=\"activate\"><a href=\"/wp-admin/plugins.php?action=activate&amp;plugin=")
			b.WriteString(html.EscapeString(url.QueryEscape(p.File)))
			b.WriteString("&amp;_wpnonce=")
			b.WriteString(html.EscapeString(nonce))
			b.WriteString("\">Activate</a></span>")
		}
		b.WriteString("</div>")
		b.WriteString("</td><td class=\"column-description desc\"><div class=\"plugin-description\"><p>")
		b.WriteString(html.EscapeString(p.Description))
		b.WriteString("</p></div>")
		// Plugin header meta line (P9.5): Version | By Author | Visit plugin site.
		b.WriteString("<div class=\"plugin-version-author-uri\">")
		var metaParts []string
		if p.Version != "" {
			metaParts = append(metaParts, "Version "+html.EscapeString(p.Version))
		}
		if p.Author != "" {
			author := html.EscapeString(p.Author)
			if p.AuthorURI != "" {
				author = `<a href="` + html.EscapeString(p.AuthorURI) + `">` + author + `</a>`
			}
			metaParts = append(metaParts, "By "+author)
		}
		if p.PluginURI != "" {
			metaParts = append(metaParts, `<a href="`+html.EscapeString(p.PluginURI)+`" target="_blank">Visit plugin site</a>`)
		}
		b.WriteString(strings.Join(metaParts, " | "))
		b.WriteString("</div>")
		if p.RequiresWP != "" || p.RequiresPHP != "" {
			b.WriteString("<div class=\"plugin-requirements\">")
			var reqs []string
			if p.RequiresWP != "" {
				reqs = append(reqs, "Requires WordPress "+html.EscapeString(p.RequiresWP))
			}
			if p.RequiresPHP != "" {
				reqs = append(reqs, "Requires PHP "+html.EscapeString(p.RequiresPHP))
			}
			b.WriteString(strings.Join(reqs, " | "))
			b.WriteString("</div>")
		}
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	writeAdminPage(w, c, newAdminChrome(c, "Plugins", "plugins-php"), b.String())
}

// handlePluginAction activates/deactivates a plugin via GET or POST (P9.4).
// Returns a query fragment notice key ("activate" / "deactivate") on success.
func handlePluginAction(r *http.Request, c *wp.Context) (string, error) {
	_ = r.ParseForm()
	action := strings.TrimSpace(r.FormValue("action"))
	if action == "" {
		action = strings.TrimSpace(r.URL.Query().Get("action"))
	}
	pluginFile := strings.TrimSpace(r.FormValue("plugin"))
	if pluginFile == "" {
		pluginFile = strings.TrimSpace(r.URL.Query().Get("plugin"))
	}
	if action == "" || pluginFile == "" {
		return "", nil
	}
	nonce := r.FormValue("_wpnonce")
	if nonce == "" {
		nonce = r.URL.Query().Get("_wpnonce")
	}
	switch action {
	case "activate":
		if c.VerifyNonce(nonce, "activate-plugin_"+pluginFile) == 0 {
			return "", fmt.Errorf("invalid nonce")
		}
		if err := plugins.ActivatePlugin(c, pluginFile); err != nil {
			return "", err
		}
		return "activate=true", nil
	case "deactivate":
		if c.VerifyNonce(nonce, "deactivate-plugin_"+pluginFile) == 0 {
			return "", fmt.Errorf("invalid nonce")
		}
		if err := plugins.DeactivatePlugin(c, pluginFile); err != nil {
			return "", err
		}
		return "deactivate=true", nil
	default:
		return "", nil
	}
}

func (h *Handler) serveProfile(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		http.Error(w, "Unauthorized\n", http.StatusUnauthorized)
		return
	}
	updated := false
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		if c.VerifyNonce(r.FormValue("_wpnonce"), "update-user_"+strconv.FormatInt(c.CurrentUser.ID, 10)) == 0 {
			http.Error(w, "Invalid nonce\n", http.StatusForbidden)
			return
		}
		displayName := strings.TrimSpace(r.FormValue("display_name"))
		if displayName == "" {
			displayName = c.CurrentUser.Login
		}
		if err := c.UpdateDisplayName(c.CurrentUser.ID, displayName); err != nil {
			http.Error(w, "profile: "+err.Error()+"\n", http.StatusInternalServerError)
			return
		}
		updated = true
	}

	user := c.CurrentUser
	login := user.Login
	email := user.Email
	display := user.DisplayName
	nonce := c.CreateNonce("update-user_" + strconv.FormatInt(user.ID, 10))

	var b strings.Builder
	b.WriteString("<h1>Profile</h1>\n")
	if updated {
		b.WriteString("<div id=\"message\" class=\"updated notice is-dismissible\"><p><strong>Profile updated.</strong></p></div>\n")
	}
	b.WriteString("<form id=\"your-profile\" method=\"post\" action=\"/wp-admin/profile.php\">\n")
	b.WriteString("<input type=\"hidden\" id=\"_wpnonce\" name=\"_wpnonce\" value=\"")
	b.WriteString(html.EscapeString(nonce))
	b.WriteString("\" />\n")
	b.WriteString("<h2>Personal Options</h2>\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr><th><label for=\"user_login\">Username</label></th><td><input type=\"text\" name=\"user_login\" id=\"user_login\" value=\"")
	b.WriteString(html.EscapeString(login))
	b.WriteString("\" disabled=\"disabled\" class=\"regular-text\"></td></tr>\n")
	b.WriteString("<tr><th><label for=\"email\">Email</label></th><td><input type=\"email\" name=\"email\" id=\"email\" value=\"")
	b.WriteString(html.EscapeString(email))
	b.WriteString("\" class=\"regular-text\"></td></tr>\n")
	b.WriteString("<tr><th><label for=\"display_name\">Display name</label></th><td><input type=\"text\" name=\"display_name\" id=\"display_name\" value=\"")
	b.WriteString(html.EscapeString(display))
	b.WriteString("\" class=\"regular-text\"></td></tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Update Profile\"></p>\n")
	b.WriteString("</form>\n")
	writeAdminPage(w, c, newAdminChrome(c, "Profile", "profile-php"), b.String())
}

func (h *Handler) serveUpload(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	items, err := recentMedia(c, 20)
	if err != nil {
		http.Error(w, "media: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	// P7.42: grid is the WP default; ?mode=list keeps the list table.
	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode != "list" && mode != "grid" {
		mode = mediaLibraryMode(c)
	}
	if mode == "grid" {
		writeHTML(w, http.StatusOK, renderMediaGrid(blogname, items, c))
		return
	}
	writeHTML(w, http.StatusOK, renderMediaList(blogname, items, c))
}

func mediaLibraryMode(c *wp.Context) string {
	if c != nil && c.CurrentUser != nil && c.Meta != nil {
		if v, ok := c.Meta.GetMetadata(wp.MetaUser, c.CurrentUser.ID, "media_library_mode", true).(string); ok {
			v = strings.TrimSpace(v)
			if v == "list" || v == "grid" {
				return v
			}
		}
	}
	return "grid"
}

func mediaViewSwitcher(mode string) string {
	var b strings.Builder
	b.WriteString(`<div class="view-switch media-grid-view-switch">`)
	gridClass := "view-grid"
	listClass := "view-list"
	if mode == "grid" {
		gridClass += " current"
	} else {
		listClass += " current"
	}
	b.WriteString(`<a href="/wp-admin/upload.php?mode=grid" class="`)
	b.WriteString(gridClass)
	b.WriteString(`" aria-label="Grid view"><span class="screen-reader-text">Grid view</span></a>`)
	b.WriteString(`<a href="/wp-admin/upload.php?mode=list" class="`)
	b.WriteString(listClass)
	b.WriteString(`" aria-label="List view"><span class="screen-reader-text">List view</span></a>`)
	b.WriteString(`</div>`)
	return b.String()
}

func renderMediaGrid(blogname string, items []mediaRow, c *wp.Context) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Media Library &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui upload-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\" id=\"wp-media-grid\" data-search=\"\">\n")
	b.WriteString("<h1 class=\"wp-heading-inline\">Media Library</h1>\n")
	b.WriteString("<a class=\"page-title-action\" href=\"/wp-admin/media-new.php\">Add Media File</a>\n")
	b.WriteString(mediaViewSwitcher("grid"))
	b.WriteString("<hr class=\"wp-header-end\">\n")
	b.WriteString("<div class=\"notice notice-error hide-if-js\"><p>The grid view for the Media Library requires JavaScript. ")
	b.WriteString(`<a href="/wp-admin/upload.php?mode=list">Switch to the list view</a>.</p></div>` + "\n")
	b.WriteString("<ul class=\"attachments ui-sortable ui-sortable-disabled\" tabindex=\"-1\">\n")
	if len(items) == 0 {
		b.WriteString("<li class=\"no-media\">No media items found.</li>\n")
	}
	for _, m := range items {
		b.WriteString(`<li class="attachment save-ready" id="post-`)
		b.WriteString(strconv.FormatInt(m.ID, 10))
		b.WriteString(`" data-id="`)
		b.WriteString(strconv.FormatInt(m.ID, 10))
		b.WriteString(`"><div class="attachment-preview"><div class="thumbnail"><div class="filename"><div>`)
		b.WriteString(html.EscapeString(m.Title))
		b.WriteString(`</div></div></div></div>`)
		b.WriteString(`<a class="edit" href="/wp-admin/post.php?post=`)
		b.WriteString(strconv.FormatInt(m.ID, 10))
		b.WriteString(`&amp;action=edit">Edit</a></li>` + "\n")
	}
	b.WriteString("</ul>\n</div>\n</body>\n</html>\n")
	return b.String()
}

func renderMediaList(blogname string, items []mediaRow, c *wp.Context) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Media Library &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui upload-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1 class=\"wp-heading-inline\">Media Library</h1>\n")
	b.WriteString("<a class=\"page-title-action\" href=\"/wp-admin/media-new.php\">Add New Media File</a>\n")
	b.WriteString(mediaViewSwitcher("list"))
	b.WriteString("<hr class=\"wp-header-end\">\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped media\">\n")
	b.WriteString("<thead><tr><th scope=\"col\" class=\"column-title\">File</th><th scope=\"col\" class=\"column-mime\">Type</th></tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	if len(items) == 0 {
		b.WriteString("<tr class=\"no-items\"><td class=\"colspanchange\" colspan=\"2\">No media items found.</td></tr>\n")
	}
	for _, m := range items {
		b.WriteString("<tr id=\"post-")
		b.WriteString(strconv.FormatInt(m.ID, 10))
		b.WriteString("\"><td class=\"title column-title\"><strong><a class=\"row-title\" href=\"/wp-admin/post.php?post=")
		b.WriteString(strconv.FormatInt(m.ID, 10))
		b.WriteString("&amp;action=edit\">")
		b.WriteString(html.EscapeString(m.Title))
		b.WriteString("</a></strong>")
		b.WriteString("<div class=\"row-actions\"><span class=\"edit\"><a href=\"/wp-admin/post.php?post=")
		b.WriteString(strconv.FormatInt(m.ID, 10))
		b.WriteString("&amp;action=edit\">Edit</a></span></div></td><td class=\"column-mime\">")
		b.WriteString(html.EscapeString(m.Mime))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n</div>\n</body>\n</html>\n")
	return b.String()
}

// serveMediaNew is the Upload New Media form stub (P7.27 / media-new.php).
func (h *Handler) serveMediaNew(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		http.Error(w, "Unauthorized\n", http.StatusUnauthorized)
		return
	}
	if !c.UserCan(c.CurrentUser, "upload_files") && !c.UserCan(c.CurrentUser, "edit_posts") {
		http.Error(w, "Sorry, you are not allowed to upload files.\n", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		h.handleMediaNewUpload(w, r, c)
		return
	}
	blogname := optionString(c, "blogname", "GoPress")
	nonce := c.CreateNonce("media-form")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Upload New Media &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Upload New Media</h1>\n")
	b.WriteString("<form enctype=\"multipart/form-data\" method=\"post\" action=\"/wp-admin/media-new.php\" class=\"wp-upload-form media-upload-form html-uploader\" id=\"file-form\">\n")
	b.WriteString("<input type=\"hidden\" id=\"_wpnonce\" name=\"_wpnonce\" value=\"")
	b.WriteString(html.EscapeString(nonce))
	b.WriteString("\" />\n")
	b.WriteString("<input type=\"hidden\" name=\"post_id\" id=\"post_id\" value=\"0\" />\n")
	b.WriteString("<p><label for=\"async-upload\">Choose a file</label>\n")
	b.WriteString("<input type=\"file\" id=\"async-upload\" name=\"async-upload\" /></p>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"html-upload\" id=\"html-upload\" class=\"button\" value=\"Upload\" /></p>\n")
	b.WriteString("</form>\n")
	b.WriteString("<p><a href=\"/wp-admin/upload.php\">&larr; Back to Media Library</a></p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) handleMediaNewUpload(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "No data supplied\n", http.StatusBadRequest)
		return
	}
	if c.VerifyNonce(r.FormValue("_wpnonce"), "media-form") == 0 {
		http.Error(w, "Invalid nonce\n", http.StatusForbidden)
		return
	}
	file, hdr, err := r.FormFile("async-upload")
	if err != nil {
		file, hdr, err = r.FormFile("file")
	}
	if err != nil {
		http.Error(w, "No data supplied\n", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if _, err := wp.SaveUploadedMedia(c, hdr.Filename, file, hdr.Size); err != nil {
		http.Error(w, err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/wp-admin/upload.php", http.StatusFound)
}

// serveAsyncUpload handles multipart POST /wp-admin/async-upload.php (P7.7).
func (h *Handler) serveAsyncUpload(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed\n", http.StatusMethodNotAllowed)
		return
	}
	if c == nil || c.CurrentUser == nil {
		http.Error(w, "Unauthorized\n", http.StatusUnauthorized)
		return
	}
	if !c.UserCan(c.CurrentUser, "upload_files") && !c.UserCan(c.CurrentUser, "edit_posts") {
		http.Error(w, "Forbidden\n", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "No data supplied\n", http.StatusBadRequest)
		return
	}
	file, hdr, err := r.FormFile("async-upload")
	if err != nil {
		// WP also accepts "file" in some clients.
		file, hdr, err = r.FormFile("file")
	}
	if err != nil {
		http.Error(w, "No data supplied\n", http.StatusBadRequest)
		return
	}
	defer file.Close()
	up, err := wp.SaveUploadedMedia(c, hdr.Filename, file, hdr.Size)
	if err != nil {
		http.Error(w, err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success":true,"data":{"id":%d,"url":%q,"filename":%q}}`+"\n",
		up.ID, up.URL, up.RelPath)
}

func (h *Handler) serveEditComments(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P7.43: bulk actions (approve / unapprove / spam / trash / delete).
	if r.Method == http.MethodPost {
		if c == nil || c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "moderate_comments") {
			http.Error(w, "Sorry, you are not allowed to edit comments.\n", http.StatusForbidden)
			return
		}
		_ = r.ParseForm()
		action := strings.TrimSpace(r.FormValue("action"))
		if action == "" || action == "-1" {
			action = strings.TrimSpace(r.FormValue("action2"))
		}
		ids := r.Form["delete_comments[]"]
		if len(ids) == 0 {
			ids = r.Form["delete_comments"]
		}
		if len(ids) == 0 {
			if raw := strings.TrimSpace(r.FormValue("ids")); raw != "" {
				ids = strings.Split(raw, ",")
			}
		}
		counts := map[string]int{}
		for _, raw := range ids {
			cid, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if cid <= 0 {
				continue
			}
			switch action {
			case "approve":
				if updateCommentApproved(c, cid, "1") == nil {
					counts["approved"]++
				}
			case "unapprove":
				if updateCommentApproved(c, cid, "0") == nil {
					counts["unapproved"]++
				}
			case "spam":
				if updateCommentApproved(c, cid, "spam") == nil {
					counts["spammed"]++
				}
			case "trash":
				if updateCommentApproved(c, cid, "trash") == nil {
					counts["trashed"]++
				}
			case "delete":
				if deleteAdminComment(c, cid) == nil {
					counts["deleted"]++
				}
			}
		}
		loc := "/wp-admin/edit-comments.php"
		sep := "?"
		for k, n := range counts {
			if n <= 0 {
				continue
			}
			loc += sep + k + "=" + strconv.Itoa(n)
			sep = "&"
		}
		if sep == "?" {
			loc += "?bulk=1"
		}
		http.Redirect(w, r, loc, http.StatusFound)
		return
	}

	blogname := optionString(c, "blogname", "GoPress")
	items, err := recentComments(c, 50)
	if err != nil {
		http.Error(w, "comments: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<h1>Comments</h1>\n")
	if n := r.URL.Query().Get("approved"); n != "" {
		b.WriteString(`<div id="message" class="updated notice is-dismissible"><p>`)
		b.WriteString(html.EscapeString(n))
		b.WriteString(` comment(s) approved.</p></div>` + "\n")
	}
	if n := r.URL.Query().Get("spammed"); n != "" {
		b.WriteString(`<div id="message" class="updated notice is-dismissible"><p>`)
		b.WriteString(html.EscapeString(n))
		b.WriteString(` comment(s) marked as spam.</p></div>` + "\n")
	}
	if n := r.URL.Query().Get("trashed"); n != "" {
		b.WriteString(`<div id="message" class="updated notice is-dismissible"><p>`)
		b.WriteString(html.EscapeString(n))
		b.WriteString(` comment(s) moved to the Trash.</p></div>` + "\n")
	}
	if n := r.URL.Query().Get("deleted"); n != "" {
		b.WriteString(`<div id="message" class="updated notice is-dismissible"><p>`)
		b.WriteString(html.EscapeString(n))
		b.WriteString(` comment(s) permanently deleted.</p></div>` + "\n")
	}
	if n := r.URL.Query().Get("unapproved"); n != "" {
		b.WriteString(`<div id="message" class="updated notice is-dismissible"><p>`)
		b.WriteString(html.EscapeString(n))
		b.WriteString(` comment(s) unapproved.</p></div>` + "\n")
	}
	b.WriteString(`<form method="post" action="/wp-admin/edit-comments.php" id="comments-form">` + "\n")
	b.WriteString(`<div class="tablenav top"><div class="alignleft actions bulkactions">`)
	b.WriteString(`<label for="bulk-action-selector-top" class="screen-reader-text">Select bulk action</label>`)
	b.WriteString(`<select name="action" id="bulk-action-selector-top">`)
	b.WriteString(`<option value="-1">Bulk actions</option>`)
	b.WriteString(`<option value="unapprove">Unapprove</option>`)
	b.WriteString(`<option value="approve">Approve</option>`)
	b.WriteString(`<option value="spam">Mark as spam</option>`)
	b.WriteString(`<option value="trash">Move to Trash</option>`)
	b.WriteString(`<option value="delete">Delete permanently</option>`)
	b.WriteString(`</select> `)
	b.WriteString(`<input type="submit" id="doaction" class="button action" value="Apply" />`)
	b.WriteString(`</div></div>` + "\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped table-view-list comments\">\n")
	b.WriteString("<thead><tr>")
	b.WriteString(`<td id="cb" class="manage-column column-cb check-column"><input id="cb-select-all-1" type="checkbox" /></td>`)
	b.WriteString("<th scope=\"col\" class=\"column-author\">Author</th>")
	b.WriteString("<th scope=\"col\" class=\"column-comment\">Comment</th>")
	b.WriteString("</tr></thead>\n")
	b.WriteString("<tbody id=\"the-comment-list\">\n")
	if len(items) == 0 {
		b.WriteString("<tr class=\"no-items\"><td class=\"colspanchange\" colspan=\"3\">No comments found.</td></tr>\n")
	}
	for _, item := range items {
		b.WriteString("<tr id=\"comment-")
		b.WriteString(strconv.FormatInt(item.ID, 10))
		b.WriteString(`" class="comment">`)
		b.WriteString(`<th scope="row" class="check-column"><input type="checkbox" name="delete_comments[]" value="`)
		b.WriteString(strconv.FormatInt(item.ID, 10))
		b.WriteString(`" /></th>`)
		b.WriteString("<td class=\"author column-author\">")
		b.WriteString(html.EscapeString(item.Author))
		b.WriteString("</td><td class=\"comment column-comment\"><p>")
		b.WriteString(html.EscapeString(item.Content))
		b.WriteString(`</p><div class="row-actions"><span class="edit"><a href="/wp-admin/comment.php?action=editcomment&amp;c=`)
		b.WriteString(strconv.FormatInt(item.ID, 10))
		b.WriteString(`">Edit</a></span></div></td></tr>` + "\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString(`<div class="tablenav bottom"><div class="alignleft actions bulkactions">`)
	b.WriteString(`<label for="bulk-action-selector-bottom" class="screen-reader-text">Select bulk action</label>`)
	b.WriteString(`<select name="action2" id="bulk-action-selector-bottom">`)
	b.WriteString(`<option value="-1">Bulk actions</option>`)
	b.WriteString(`<option value="unapprove">Unapprove</option>`)
	b.WriteString(`<option value="approve">Approve</option>`)
	b.WriteString(`<option value="spam">Mark as spam</option>`)
	b.WriteString(`<option value="trash">Move to Trash</option>`)
	b.WriteString(`<option value="delete">Delete permanently</option>`)
	b.WriteString(`</select> `)
	b.WriteString(`<input type="submit" id="doaction2" class="button action" value="Apply" />`)
	b.WriteString(`</div></div></form>` + "\n")
	_ = blogname
	writeAdminPage(w, c, newAdminChrome(c, "Comments", "edit-comments-php"), b.String())
}

// serveCommentModerate is the comment.php moderation / edit stub (P7.30).
func (h *Handler) serveCommentModerate(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	if action == "" {
		action = strings.TrimSpace(r.FormValue("action"))
	}
	switch action {
	case "cdc":
		action = "delete"
	case "mac":
		action = "approve"
	}
	if dt := r.URL.Query().Get("dt"); dt != "" {
		switch dt {
		case "spam":
			action = "spam"
		case "trash":
			action = "trash"
		}
	}
	cid, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("c")), 10, 64)
	if cid <= 0 {
		http.Redirect(w, r, "/wp-admin/edit-comments.php?error=1", http.StatusFound)
		return
	}
	if c == nil || c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "moderate_comments") {
		http.Error(w, "Sorry, you are not allowed to edit this comment.\n", http.StatusForbidden)
		return
	}
	cm, err := loadAdminComment(c, cid)
	if err != nil {
		http.Error(w, "comment: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if cm == nil {
		http.Redirect(w, r, "/wp-admin/edit-comments.php?error=1", http.StatusFound)
		return
	}

	// Apply moderation actions (approve/spam/trash/delete).
	if r.Method == http.MethodPost || action == "approvecomment" || action == "deletecomment" ||
		action == "spamcomment" || action == "trashcomment" {
		switch action {
		case "approve", "approvecomment":
			_ = updateCommentApproved(c, cid, "1")
			http.Redirect(w, r, "/wp-admin/edit-comments.php?approved=1", http.StatusFound)
			return
		case "spam", "spamcomment":
			_ = updateCommentApproved(c, cid, "spam")
			http.Redirect(w, r, "/wp-admin/edit-comments.php?spammed=1", http.StatusFound)
			return
		case "trash", "trashcomment":
			_ = updateCommentApproved(c, cid, "trash")
			http.Redirect(w, r, "/wp-admin/edit-comments.php?trashed=1", http.StatusFound)
			return
		case "delete", "deletecomment":
			_ = deleteAdminComment(c, cid)
			http.Redirect(w, r, "/wp-admin/edit-comments.php?deleted=1", http.StatusFound)
			return
		}
	}

	heading := "Edit Comment"
	caution := ""
	button := ""
	formAction := ""
	switch action {
	case "approve":
		heading = "Moderate Comment"
		caution = "You are about to approve the following comment:"
		button = "Approve comment"
		formAction = "approvecomment"
	case "spam":
		heading = "Moderate Comment"
		caution = "You are about to mark the following comment as spam:"
		button = "Mark as spam"
		formAction = "spamcomment"
	case "trash":
		heading = "Moderate Comment"
		caution = "You are about to move the following comment to the Trash:"
		button = "Move to Trash"
		formAction = "trashcomment"
	case "delete":
		heading = "Moderate Comment"
		caution = "You are about to delete the following comment:"
		button = "Permanently delete comment"
		formAction = "deletecomment"
	default:
		action = "editcomment"
	}

	var b strings.Builder
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(heading))
	b.WriteString("</h1>\n")
	b.WriteString("<p class=\"description\">GoPress comment.php moderation stub (P7.30).</p>\n")
	if caution != "" {
		b.WriteString("<div id=\"message\" class=\"notice notice-warning\"><p><strong>Caution:</strong> ")
		b.WriteString(html.EscapeString(caution))
		b.WriteString("</p></div>\n")
	}
	b.WriteString("<table class=\"form-table comment-ays\">\n")
	b.WriteString("<tr><th scope=\"row\">Author</th><td>")
	b.WriteString(html.EscapeString(cm.Author))
	b.WriteString("</td></tr>\n")
	if cm.Email != "" {
		b.WriteString("<tr><th scope=\"row\">Email</th><td>")
		b.WriteString(html.EscapeString(cm.Email))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("<tr><th scope=\"row\">Comment</th><td><p>")
	b.WriteString(html.EscapeString(cm.Content))
	b.WriteString("</p></td></tr>\n")
	b.WriteString("<tr><th scope=\"row\">Status</th><td>")
	b.WriteString(html.EscapeString(commentApprovedLabel(cm.Approved)))
	b.WriteString("</td></tr>\n")
	b.WriteString("</table>\n")
	if formAction != "" {
		b.WriteString("<form method=\"post\" action=\"/wp-admin/comment.php?c=")
		b.WriteString(strconv.FormatInt(cid, 10))
		b.WriteString("&amp;action=")
		b.WriteString(html.EscapeString(formAction))
		b.WriteString("\">\n")
		b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" class=\"button button-primary\" value=\"")
		b.WriteString(html.EscapeString(button))
		b.WriteString("\" /> ")
		b.WriteString("<a class=\"button\" href=\"/wp-admin/edit-comments.php\">Cancel</a></p>\n")
		b.WriteString("</form>\n")
	} else {
		b.WriteString("<p class=\"submit\">")
		b.WriteString(`<a class="button" href="/wp-admin/comment.php?action=approve&amp;c=`)
		b.WriteString(strconv.FormatInt(cid, 10))
		b.WriteString(`">Approve</a> `)
		b.WriteString(`<a class="button" href="/wp-admin/comment.php?action=spam&amp;c=`)
		b.WriteString(strconv.FormatInt(cid, 10))
		b.WriteString(`">Spam</a> `)
		b.WriteString(`<a class="button" href="/wp-admin/comment.php?action=trash&amp;c=`)
		b.WriteString(strconv.FormatInt(cid, 10))
		b.WriteString(`">Trash</a> `)
		b.WriteString(`<a class="button" href="/wp-admin/edit-comments.php">Back to Comments</a>`)
		b.WriteString("</p>\n")
	}
	writeAdminPage(w, c, newAdminChrome(c, heading, "comment-php"), b.String())
}

type adminComment struct {
	ID       int64
	Author   string
	Email    string
	Content  string
	Approved string
	PostID   int64
}

func loadAdminComment(c *wp.Context, id int64) (*adminComment, error) {
	if c == nil || c.DB == nil || id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID, comment_author, comment_author_email, comment_content, comment_approved, comment_post_ID "+
			"FROM "+c.DB.Comments()+" WHERE comment_ID = %d LIMIT 1",
		id,
	)
	if err != nil {
		return nil, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return &adminComment{
		ID:       asInt64(row["comment_ID"]),
		Author:   asString(row["comment_author"]),
		Email:    asString(row["comment_author_email"]),
		Content:  asString(row["comment_content"]),
		Approved: asString(row["comment_approved"]),
		PostID:   asInt64(row["comment_post_ID"]),
	}, nil
}

func updateCommentApproved(c *wp.Context, id int64, approved string) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("invalid comment")
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Comments()+" SET comment_approved = %s WHERE comment_ID = %d",
		approved, id,
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func deleteAdminComment(c *wp.Context, id int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("invalid comment")
	}
	q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Comments()+" WHERE comment_ID = %d", id)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func commentApprovedLabel(approved string) string {
	switch approved {
	case "1":
		return "Approved"
	case "0", "hold":
		return "Pending"
	case "spam":
		return "Spam"
	case "trash":
		return "Trash"
	default:
		return approved
	}
}

func (h *Handler) serveThemes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	stylesheet := optionString(c, "stylesheet", "twentytwentytwo")
	template := optionString(c, "template", stylesheet)
	absPath := ""
	if c != nil && c.Config != nil {
		absPath = c.Config.ABSPATH
	}
	themes, err := theme.ListInstalledThemes(absPath)
	if err != nil {
		http.Error(w, "themes: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	// Ensure the active theme appears even if style.css parse failed.
	hasActive := false
	for _, th := range themes {
		if th.Slug == stylesheet {
			hasActive = true
			break
		}
	}
	if !hasActive && stylesheet != "" {
		themes = append([]theme.ThemeData{{
			Slug:     stylesheet,
			Name:     stylesheet,
			Template: template,
		}}, themes...)
	}
	// Active theme first (WP themes.php order).
	for i, th := range themes {
		if th.Slug == stylesheet {
			themes = append(append([]theme.ThemeData{th}, themes[:i]...), themes[i+1:]...)
			break
		}
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Themes &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui themes-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Themes</h1>\n")
	b.WriteString("<div class=\"theme-browser\">\n")
	for _, th := range themes {
		active := th.Slug == stylesheet
		b.WriteString("<div class=\"theme")
		if active {
			b.WriteString(" active")
		}
		b.WriteString("\" data-slug=\"")
		b.WriteString(html.EscapeString(th.Slug))
		b.WriteString("\">\n")
		b.WriteString("<div class=\"theme-id-container\">\n")
		b.WriteString("<h2 class=\"theme-name\" id=\"")
		b.WriteString(html.EscapeString(th.Slug))
		b.WriteString("-name\">")
		if active {
			b.WriteString("<span>Active:</span> ")
		}
		name := th.Name
		if name == "" {
			name = th.Slug
		}
		b.WriteString(html.EscapeString(name))
		b.WriteString("</h2>\n")
		if th.Author != "" {
			b.WriteString("<div class=\"theme-author\">By ")
			b.WriteString(html.EscapeString(th.Author))
			b.WriteString("</div>\n")
		}
		if th.Version != "" {
			b.WriteString("<p class=\"theme-version\">Version: ")
			b.WriteString(html.EscapeString(th.Version))
			b.WriteString("</p>\n")
		}
		if th.Description != "" {
			b.WriteString("<p class=\"theme-description\">")
			b.WriteString(html.EscapeString(th.Description))
			b.WriteString("</p>\n")
		}
		tpl := th.Template
		if active && template != "" {
			tpl = template
		}
		if tpl != "" {
			b.WriteString("<p class=\"theme-template\">Template: ")
			b.WriteString(html.EscapeString(tpl))
			b.WriteString("</p>\n")
		}
		b.WriteString("<button type=\"button\" class=\"button button-secondary theme-detail\" aria-label=\"View Theme Details for ")
		b.WriteString(html.EscapeString(name))
		b.WriteString("\">Theme Details</button>\n")
		b.WriteString("</div>\n</div>\n")
	}
	b.WriteString("</div>\n")
	b.WriteString("<div class=\"theme-overlay theme-overlay-initial\" tabindex=\"0\" role=\"dialog\" aria-label=\"Theme Details\"></div>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) serveSiteEditor(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	stylesheet := optionString(c, "stylesheet", "twentytwentytwo")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Editor &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<script>var wpApiSettings = {\"root\":\"/wp-json/\",\"nonce\":\"")
	b.WriteString(html.EscapeString(c.CreateNonce("wp_rest")))
	b.WriteString("\"};</script>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui block-editor-page site-editor\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div id=\"site-editor\" class=\"edit-site\">\n")
	b.WriteString("<h1>Site Editor</h1>\n")
	b.WriteString("<p>Editing theme: ")
	b.WriteString(html.EscapeString(stylesheet))
	b.WriteString("</p>\n")
	b.WriteString("<p>GoPress site editor stub (P7.5) — full block canvas lands later.</p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveNavMenus is a classic menus admin stub (P7.9) with location save (P7.35).
func (h *Handler) serveNavMenus(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	menus, err := listNavMenus(c)
	if err != nil {
		http.Error(w, "menus: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	note := ""
	locationsScreen := r.URL.Query().Get("action") == "locations" || r.Method == http.MethodPost
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		// Save theme menu location assignments (nav-menus.php Manage Locations, P7.35).
		registered := wp.RegisteredNavMenus(c)
		for loc := range registered {
			raw := strings.TrimSpace(r.Form.Get("menu-locations[" + loc + "]"))
			id, _ := strconv.ParseInt(raw, 10, 64)
			if err := wp.SetNavMenuLocation(c, loc, id); err != nil {
				note = "Failed to save menu locations: " + err.Error()
				break
			}
		}
		if note == "" {
			note = "Menu locations updated."
			locationsScreen = true
		}
	}
	locs := wp.GetNavMenuLocations(c)
	registered := wp.RegisteredNavMenus(c)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Menus &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui nav-menus-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\" id=\"menu-management\">\n")
	b.WriteString("<h1>Menus</h1>\n")
	if note != "" {
		b.WriteString("<div id=\"message\" class=\"updated notice is-dismissible\"><p>")
		b.WriteString(html.EscapeString(note))
		b.WriteString("</p></div>\n")
	}
	b.WriteString("<h2 class=\"nav-tab-wrapper wp-clearfix\">\n")
	editClass, locClass := " nav-tab-active", ""
	if locationsScreen {
		editClass, locClass = "", " nav-tab-active"
	}
	b.WriteString("<a href=\"/wp-admin/nav-menus.php\" class=\"nav-tab")
	b.WriteString(editClass)
	b.WriteString("\">Edit Menus</a>\n")
	b.WriteString("<a href=\"/wp-admin/nav-menus.php?action=locations\" class=\"nav-tab")
	b.WriteString(locClass)
	b.WriteString("\">Manage Locations</a>\n")
	b.WriteString("</h2>\n")

	if locationsScreen {
		b.WriteString("<div id=\"menu-locations-wrap\">\n")
		b.WriteString("<form method=\"post\" action=\"/wp-admin/nav-menus.php?action=locations\">\n")
		b.WriteString("<table class=\"widefat fixed\" id=\"menu-locations-table\">\n")
		b.WriteString("<thead><tr><th scope=\"col\">Theme Location</th><th scope=\"col\">Assigned Menu</th></tr></thead>\n")
		b.WriteString("<tbody class=\"menu-locations\">\n")
		// Stable order: primary then footer then others.
		order := []string{"primary", "footer"}
		seen := map[string]bool{}
		for _, loc := range order {
			if _, ok := registered[loc]; ok {
				writeNavMenuLocationRow(&b, loc, registered[loc], menus, locs[loc])
				seen[loc] = true
			}
		}
		for loc, desc := range registered {
			if seen[loc] {
				continue
			}
			writeNavMenuLocationRow(&b, loc, desc, menus, locs[loc])
		}
		b.WriteString("</tbody>\n</table>\n")
		b.WriteString("<p class=\"button-controls wp-clearfix\">")
		b.WriteString("<input type=\"submit\" name=\"nav-menu-locations\" id=\"nav-menu-locations\" class=\"button button-primary left\" value=\"Save Changes\">")
		b.WriteString("</p>\n</form>\n</div>\n")
	} else {
		b.WriteString("<p>GoPress nav-menus stub (P7.9) — drag-and-drop editor lands later.</p>\n")
		b.WriteString("<div id=\"nav-menus-frame\" class=\"wp-clearfix\">\n")
		b.WriteString("<ul class=\"menu\">\n")
		if len(menus) == 0 {
			b.WriteString("<li class=\"no-items\">No menus found.</li>\n")
		} else {
			for _, m := range menus {
				b.WriteString("<li class=\"menu-item\" data-menu-id=\"")
				b.WriteString(strconv.FormatInt(m.ID, 10))
				b.WriteString("\"><strong>")
				b.WriteString(html.EscapeString(m.Name))
				b.WriteString("</strong> <span class=\"menu-slug\">")
				b.WriteString(html.EscapeString(m.Slug))
				b.WriteString("</span></li>\n")
			}
		}
		b.WriteString("</ul>\n</div>\n")
	}
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func writeNavMenuLocationRow(b *strings.Builder, loc, desc string, menus []navMenuTerm, assigned int64) {
	b.WriteString("<tr class=\"menu-locations-row\">\n")
	b.WriteString("<td><label for=\"locations-")
	b.WriteString(html.EscapeString(loc))
	b.WriteString("\"><strong>")
	b.WriteString(html.EscapeString(desc))
	b.WriteString("</strong></label><br><span class=\"theme-location-name\">")
	b.WriteString(html.EscapeString(loc))
	b.WriteString("</span></td>\n<td>\n")
	b.WriteString("<select name=\"menu-locations[")
	b.WriteString(html.EscapeString(loc))
	b.WriteString("]\" id=\"locations-")
	b.WriteString(html.EscapeString(loc))
	b.WriteString("\">\n")
	b.WriteString("<option value=\"0\">— Select a Menu —</option>\n")
	for _, m := range menus {
		b.WriteString("<option value=\"")
		b.WriteString(strconv.FormatInt(m.ID, 10))
		b.WriteString("\"")
		if m.ID == assigned {
			b.WriteString(" selected=\"selected\"")
		}
		b.WriteString(">")
		b.WriteString(html.EscapeString(m.Name))
		b.WriteString("</option>\n")
	}
	b.WriteString("</select>\n</td>\n</tr>\n")
}

type navMenuTerm struct {
	ID   int64
	Name string
	Slug string
}

func listNavMenus(c *wp.Context) ([]navMenuTerm, error) {
	if c == nil || c.DB == nil {
		return nil, nil
	}
	q := "SELECT t.term_id, t.name, t.slug FROM " + c.DB.Terms() + " t " +
		"INNER JOIN " + c.DB.TermTaxonomy() + " tt ON t.term_id = tt.term_id " +
		"WHERE tt.taxonomy = %s ORDER BY t.name ASC LIMIT 50"
	sqlStr, args, err := c.DB.Prepare(q, "nav_menu")
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	out := make([]navMenuTerm, 0, len(rows))
	for _, row := range rows {
		id, _ := strconv.ParseInt(fmt.Sprint(row["term_id"]), 10, 64)
		out = append(out, navMenuTerm{
			ID:   id,
			Name: fmt.Sprint(row["name"]),
			Slug: fmt.Sprint(row["slug"]),
		})
	}
	return out, nil
}

// serveSiteHealth is the Tools → Site Health screen stub (P8.3).
func (h *Handler) serveSiteHealth(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Site Health &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui site-health\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap health-check-header\" id=\"site-health\">\n")
	b.WriteString("<h1>Site Health</h1>\n")
	b.WriteString("<p>GoPress site-health stub — async REST checks land later.</p>\n")
	b.WriteString("<div class=\"health-check-body\">\n")
	b.WriteString("<div class=\"site-health-progress-wrapper\">\n")
	b.WriteString("<div class=\"site-health-progress\"><span class=\"screen-reader-text\">Good</span></div>\n")
	b.WriteString("</div>\n")
	b.WriteString("<ul class=\"health-check-accordion\">\n")
	b.WriteString("<li class=\"health-check-accordion-item\"><h2>Status</h2><p>All stub checks report good.</p></li>\n")
	b.WriteString("</ul>\n</div>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveTools is the Tools screen stub (P7.26).
func (h *Handler) serveTools(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// Legacy privacy redirects from tools.php?page=… (WP 5.3+).
	if page := r.URL.Query().Get("page"); page != "" {
		switch page {
		case "export_personal_data":
			http.Redirect(w, r, "/wp-admin/export-personal-data.php", http.StatusMovedPermanently)
			return
		case "remove_personal_data":
			http.Redirect(w, r, "/wp-admin/erase-personal-data.php", http.StatusMovedPermanently)
			return
		}
	}
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Tools &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Tools</h1>\n")
	b.WriteString("<p>Press This, Categories and Tags Converter, and other tools live here.</p>\n")
	b.WriteString("<p class=\"description\">GoPress tools stub — converter plugins land later.</p>\n")
	b.WriteString("<div class=\"card\">\n")
	b.WriteString("<h2 class=\"title\">Categories and Tags Converter</h2>\n")
	b.WriteString("<p>If you want to convert your categories to tags (or vice versa), use the ")
	b.WriteString("<a href=\"import.php\">Categories and Tags Converter</a> available from the Import screen.</p>\n")
	b.WriteString("</div>\n")
	b.WriteString("<p><a href=\"export.php\">Export</a> · <a href=\"import.php\">Import</a> · ")
	b.WriteString("<a href=\"site-health.php\">Site Health</a> · ")
	b.WriteString("<a href=\"export-personal-data.php\">Export Personal Data</a> · ")
	b.WriteString("<a href=\"erase-personal-data.php\">Erase Personal Data</a></p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveImport is the Tools → Import screen stub (P7.11).
func (h *Handler) serveImport(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Import &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Import</h1>\n")
	b.WriteString("<p>Choose an importer to bring content from another platform into this site.</p>\n")
	b.WriteString("<p class=\"description\">GoPress import stub — importer plugins land later.</p>\n")
	b.WriteString("<table class=\"widefat importers striped\">\n")
	b.WriteString("<thead><tr><th scope=\"col\">Importer</th><th scope=\"col\">Description</th></tr></thead>\n")
	b.WriteString("<tbody><tr class=\"no-items\"><td class=\"colspanchange\" colspan=\"2\">No importers are available.</td></tr></tbody>\n")
	b.WriteString("</table>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveExport is the Tools → Export screen stub (P7.10).
func (h *Handler) serveExport(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Export &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Export</h1>\n")
	b.WriteString("<p>When you click the button below WordPress will create an XML file of your site content.</p>\n")
	b.WriteString("<p class=\"description\">GoPress export stub — WXR download lands later.</p>\n")
	b.WriteString("<form method=\"get\" id=\"export-filters\">\n")
	b.WriteString("<fieldset><legend class=\"screen-reader-text\">Content to export</legend>\n")
	b.WriteString("<p><label><input type=\"radio\" name=\"content\" value=\"all\" checked=\"checked\"> All content</label></p>\n")
	b.WriteString("<p><label><input type=\"radio\" name=\"content\" value=\"posts\"> Posts</label></p>\n")
	b.WriteString("<p><label><input type=\"radio\" name=\"content\" value=\"pages\"> Pages</label></p>\n")
	b.WriteString("<p><label><input type=\"radio\" name=\"content\" value=\"attachment\"> Media</label></p>\n")
	b.WriteString("</fieldset>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Download Export File\" disabled=\"disabled\"></p>\n")
	b.WriteString("</form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveExportPersonalData is a privacy tools stub (P8.2) with file generation (P8.4),
// download wiring (P8.6), and UI polish (P7.39).
func (h *Handler) serveExportPersonalData(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method == http.MethodGet && r.URL.Query().Get("action") == "download" {
		h.servePersonalDataExportDownload(w, r, c)
		return
	}
	note := "This tool helps site owners comply with local laws and regulations by exporting known data for a given user."
	var downloadLinks string
	pendingEmail := ""
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		email := strings.TrimSpace(r.Form.Get("username_or_email_for_privacy_request"))
		if email == "" {
			email = strings.TrimSpace(r.Form.Get("username"))
		}
		if email == "" {
			email = strings.TrimSpace(r.Form.Get("email"))
		}
		action := strings.TrimSpace(r.Form.Get("action"))
		switch action {
		case "confirm_export":
			if res, err := wp.GeneratePersonalDataExportFile(c, email); err != nil {
				note = "Export generation failed: " + err.Error()
			} else {
				jsonName := filepath.Base(res.JSONPath)
				htmlName := filepath.Base(res.HTMLPath)
				jsonDL := wp.PersonalDataExportDownloadURL(jsonName)
				htmlDL := wp.PersonalDataExportDownloadURL(htmlName)
				note = "Export file generated for " + email + "."
				downloadLinks = "<p class=\"export-download-links\">" +
					"<a class=\"button button-primary\" href=\"" + html.EscapeString(jsonDL) + "\">Download JSON</a> " +
					"<a class=\"button\" href=\"" + html.EscapeString(htmlDL) + "\">Download HTML</a>" +
					"</p>\n"
			}
		default:
			// WP: send_confirmation_email checked → pending; unchecked → generate now.
			sendConfirm := r.Form.Get("send_confirmation_email") != ""
			if sendConfirm {
				pendingEmail = email
				note = "Confirmation required: confirm export for " + email + "."
			} else if res, err := wp.GeneratePersonalDataExportFile(c, email); err != nil {
				note = "Export generation failed: " + err.Error()
			} else {
				jsonName := filepath.Base(res.JSONPath)
				htmlName := filepath.Base(res.HTMLPath)
				jsonDL := wp.PersonalDataExportDownloadURL(jsonName)
				htmlDL := wp.PersonalDataExportDownloadURL(htmlName)
				note = "Export file generated for " + email + "."
				downloadLinks = "<p class=\"export-download-links\">" +
					"<a class=\"button button-primary\" href=\"" + html.EscapeString(jsonDL) + "\">Download JSON</a> " +
					"<a class=\"button\" href=\"" + html.EscapeString(htmlDL) + "\">Download HTML</a>" +
					"</p>\n"
			}
		}
	}
	h.servePrivacyExportPage(w, c, note, downloadLinks, pendingEmail)
}

func (h *Handler) servePersonalDataExportDownload(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	fileName := strings.TrimSpace(r.URL.Query().Get("file"))
	full := wp.ResolvePersonalDataExportFile(c, fileName)
	if full == "" {
		http.Error(w, "Export file not found.\n", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(full)+"\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, full)
}

func (h *Handler) servePrivacyExportPage(w http.ResponseWriter, c *wp.Context, note, downloadLinks, pendingEmail string) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Export Personal Data &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui privacy-settings export-personal-data-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap nosubsub\" id=\"privacy-settings\">\n")
	b.WriteString("<h1>Export Personal Data</h1>\n")
	b.WriteString("<p>This tool helps site owners comply with local laws and regulations by exporting known data for a given user in a .zip file.</p>\n")
	b.WriteString("<hr class=\"wp-header-end\">\n")
	b.WriteString("<p class=\"description\">")
	b.WriteString(html.EscapeString(note))
	b.WriteString("</p>\n")
	b.WriteString(downloadLinks)
	if pendingEmail != "" {
		b.WriteString("<div class=\"notice notice-warning\" id=\"export-confirm-pending\"><p>")
		b.WriteString("A confirmation email would be sent to <strong>")
		b.WriteString(html.EscapeString(pendingEmail))
		b.WriteString("</strong>. Confirm below to complete the export (P7.39 stub).</p>\n")
		b.WriteString("<form method=\"post\" action=\"/wp-admin/export-personal-data.php\" class=\"wp-privacy-export-confirm-form\">\n")
		b.WriteString("<input type=\"hidden\" name=\"action\" value=\"confirm_export\">\n")
		b.WriteString("<input type=\"hidden\" name=\"email\" value=\"")
		b.WriteString(html.EscapeString(pendingEmail))
		b.WriteString("\">\n")
		b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"confirm_export\" id=\"confirm_export\" class=\"button button-primary\" value=\"Confirm Export\"></p>\n")
		b.WriteString("</form></div>\n")
	}
	b.WriteString("<form method=\"post\" action=\"/wp-admin/export-personal-data.php\" class=\"wp-privacy-request-form\">\n")
	b.WriteString("<h2>Add Data Export Request</h2>\n")
	b.WriteString("<div class=\"wp-privacy-request-form-field\">\n")
	b.WriteString("<table class=\"form-table\">\n")
	b.WriteString("<tr><th scope=\"row\"><label for=\"username_or_email_for_privacy_request\">Username or email address</label></th>\n")
	b.WriteString("<td><input name=\"username_or_email_for_privacy_request\" type=\"text\" id=\"username_or_email_for_privacy_request\" class=\"regular-text ltr\" required></td></tr>\n")
	b.WriteString("<tr><th scope=\"row\">Confirmation email</th><td>\n")
	b.WriteString("<input type=\"checkbox\" name=\"send_confirmation_email\" id=\"send_confirmation_email\" value=\"1\" checked=\"checked\">\n")
	b.WriteString("<label for=\"send_confirmation_email\">Send personal data export confirmation email.</label>\n")
	b.WriteString("</td></tr>\n</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Send Request\"></p>\n")
	b.WriteString("</div>\n</form>\n")
	b.WriteString(renderExportPrivacyRequestsTable(c, pendingEmail))
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveErasePersonalData is a privacy tools stub (P8.2) with erasure (P8.5) and confirm (P8.8).
func (h *Handler) serveErasePersonalData(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	note := "GoPress privacy erasure stub — full request workflow lands later."
	pendingEmail := ""
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		email := strings.TrimSpace(r.Form.Get("username"))
		if email == "" {
			email = strings.TrimSpace(r.Form.Get("email"))
		}
		if email == "" {
			email = strings.TrimSpace(r.Form.Get("username_or_email_for_privacy_request"))
		}
		action := strings.TrimSpace(r.Form.Get("action"))
		switch action {
		case "confirm_erasure":
			if res, err := wp.ErasePersonalData(c, email); err != nil {
				note = "Erasure failed: " + err.Error()
			} else {
				note = res.Message
			}
		default:
			// WP: send_confirmation_email checked → pending; unchecked → confirmed/erase.
			sendConfirm := r.Form.Get("send_confirmation_email") != ""
			if sendConfirm {
				pendingEmail = email
				note = "Confirmation required: confirm erasure for " + email + "."
			} else if res, err := wp.ErasePersonalData(c, email); err != nil {
				note = "Erasure failed: " + err.Error()
			} else {
				note = res.Message
			}
		}
	}
	h.servePrivacyErasePage(w, c, note, pendingEmail)
}

func (h *Handler) servePrivacyErasePage(w http.ResponseWriter, c *wp.Context, note, pendingEmail string) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Erase Personal Data &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui privacy-settings\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\" id=\"privacy-settings\">\n")
	b.WriteString("<h1>Erase Personal Data</h1>\n")
	b.WriteString("<p>Manage requests to erase personal data.</p>\n")
	b.WriteString("<p class=\"description\">")
	b.WriteString(html.EscapeString(note))
	b.WriteString("</p>\n")
	if pendingEmail != "" {
		b.WriteString("<div class=\"notice notice-warning\" id=\"erase-confirm-pending\"><p>")
		b.WriteString("A confirmation email would be sent to <strong>")
		b.WriteString(html.EscapeString(pendingEmail))
		b.WriteString("</strong>. Confirm below to complete erasure (P8.8 stub).</p>\n")
		b.WriteString("<form method=\"post\" action=\"/wp-admin/erase-personal-data.php\" class=\"wp-privacy-erase-confirm-form\">\n")
		b.WriteString("<input type=\"hidden\" name=\"action\" value=\"confirm_erasure\">\n")
		b.WriteString("<input type=\"hidden\" name=\"email\" value=\"")
		b.WriteString(html.EscapeString(pendingEmail))
		b.WriteString("\">\n")
		b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"confirm_erasure\" id=\"confirm_erasure\" class=\"button button-primary\" value=\"Confirm Erasure\"></p>\n")
		b.WriteString("</form></div>\n")
	}
	b.WriteString("<form method=\"post\" action=\"/wp-admin/erase-personal-data.php\" class=\"wp-privacy-request-form\">\n")
	b.WriteString("<h2>Add Data Erasure Request</h2>\n")
	b.WriteString("<p><label for=\"username\">Username or email address</label><br>\n")
	b.WriteString("<input name=\"username\" type=\"text\" id=\"username\" class=\"regular-text\" required></p>\n")
	b.WriteString("<p><input type=\"checkbox\" name=\"send_confirmation_email\" id=\"send_confirmation_email\" value=\"1\" checked=\"checked\">\n")
	b.WriteString("<label for=\"send_confirmation_email\">Send personal data erasure confirmation email.</label></p>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Send Request\"></p>\n")
	b.WriteString("</form>\n")
	b.WriteString(renderErasePrivacyRequestsTable(pendingEmail))
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveOptionsPrivacy is the Settings → Privacy screen (P8.2 / P8.7 / P7.41).
// POST action=set-privacy-page saves wp_page_for_privacy_policy.
func (h *Handler) serveOptionsPrivacy(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.URL.Query().Get("tab") == "policyguide" {
		http.Redirect(w, r, "/wp-admin/privacy-policy-guide.php", http.StatusFound)
		return
	}
	note := ""
	noteType := "updated"
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		if r.Form.Get("action") == "set-privacy-page" {
			raw := strings.TrimSpace(r.Form.Get("page_for_privacy_policy"))
			id, _ := strconv.ParseInt(raw, 10, 64)
			if id < 0 {
				id = 0
			}
			prev := wp.PrivacyPolicyPageID(c)
			if c.Options != nil {
				_, _ = c.Options.UpdateOption("wp_page_for_privacy_policy", strconv.FormatInt(id, 10), "yes")
			}
			switch {
			case id > 0:
				note = "Privacy Policy page updated successfully."
			case prev > 0:
				note = "Privacy Policy page removed."
			default:
				note = "No Privacy Policy page is currently set."
				noteType = "notice-info"
			}
		}
	}

	blogname := optionString(c, "blogname", "GoPress")
	selected := wp.PrivacyPolicyPageID(c)
	pages := listPrivacyPolicyPages(c)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Privacy &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui privacy-settings\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"privacy-settings-header\">\n")
	b.WriteString("<div class=\"privacy-settings-title-section\"><h1>Privacy</h1></div>\n")
	b.WriteString("<nav class=\"privacy-settings-tabs-wrapper\" aria-label=\"Secondary menu\">\n")
	b.WriteString("<a href=\"/wp-admin/options-privacy.php\" class=\"privacy-settings-tab active\" aria-current=\"true\">Settings</a>\n")
	b.WriteString("<a href=\"/wp-admin/privacy-policy-guide.php\" class=\"privacy-settings-tab\">Policy Guide</a>\n")
	b.WriteString("</nav>\n</div>\n<hr class=\"wp-header-end\">\n")
	b.WriteString("<div class=\"wrap\" id=\"privacy-settings\">\n")
	b.WriteString("<p>Privacy settings and policy page.</p>\n")
	if note != "" {
		b.WriteString("<div id=\"message\" class=\"")
		b.WriteString(html.EscapeString(noteType))
		b.WriteString(" notice is-dismissible\"><p>")
		b.WriteString(html.EscapeString(note))
		b.WriteString("</p></div>\n")
	}
	b.WriteString("<p class=\"privacy-policy-page-link\">")
	b.WriteString(wp.PrivacyPolicyAdminSummary(c))
	b.WriteString("</p>\n")
	b.WriteString("<h2>Privacy Policy page</h2>\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n<tr>\n")
	b.WriteString("<th scope=\"row\"><label for=\"page_for_privacy_policy\">")
	if selected > 0 {
		b.WriteString("Change your Privacy Policy page")
	} else {
		b.WriteString("Select a Privacy Policy page")
	}
	b.WriteString("</label></th>\n<td>\n")
	b.WriteString("<form method=\"post\" action=\"/wp-admin/options-privacy.php\">\n")
	b.WriteString("<input type=\"hidden\" name=\"action\" value=\"set-privacy-page\" />\n")
	b.WriteString("<select name=\"page_for_privacy_policy\" id=\"page_for_privacy_policy\">\n")
	b.WriteString("<option value=\"0\">&mdash; Select &mdash;</option>\n")
	for _, p := range pages {
		b.WriteString("<option value=\"")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\"")
		if p.ID == selected {
			b.WriteString(" selected=\"selected\"")
		}
		b.WriteString(">")
		title := strings.TrimSpace(p.Title)
		if title == "" {
			title = p.Name
		}
		b.WriteString(html.EscapeString(title))
		if p.Status != "publish" {
			b.WriteString(" (")
			b.WriteString(html.EscapeString(p.Status))
			b.WriteString(")")
		}
		b.WriteString("</option>\n")
	}
	b.WriteString("</select>\n")
	b.WriteString("<input type=\"submit\" name=\"submit\" id=\"set-page\" class=\"button button-primary\" value=\"Use This Page\" />\n")
	b.WriteString("</form>\n</td>\n</tr>\n</table>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// listPrivacyPolicyPages returns draft+publish pages for the privacy dropdown (P7.41).
func listPrivacyPolicyPages(c *wp.Context) []wp.Post {
	if c == nil {
		return nil
	}
	var out []wp.Post
	for _, status := range []string{"publish", "draft"} {
		pages, err := wp.GetPosts(c, wp.QueryArgs{
			PostType:     "page",
			PostStatus:   status,
			PostsPerPage: 100,
		})
		if err != nil {
			continue
		}
		out = append(out, pages...)
	}
	return out
}

// serveRevisionCompare is the wp-admin/revision.php compare screen stub (P7.33).
func (h *Handler) serveRevisionCompare(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	revisionID := strings.TrimSpace(r.URL.Query().Get("revision"))
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	if action == "" {
		action = "view"
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Revisions &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui revisions-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Compare Revisions</h1>\n")
	b.WriteString("<div id=\"revisions\" class=\"revisions\">\n")
	b.WriteString("<div class=\"revisions-control-frame\">\n")
	b.WriteString("<div class=\"revisions-controls\">\n")
	b.WriteString("<div class=\"revisions-buttons\">\n")
	b.WriteString("<div class=\"revisions-previous\"><input class=\"button\" type=\"button\" value=\"Previous\" disabled=\"disabled\" /></div>\n")
	b.WriteString("<div class=\"revisions-next\"><input class=\"button\" type=\"button\" value=\"Next\" disabled=\"disabled\" /></div>\n")
	b.WriteString("</div>\n</div>\n</div>\n")
	b.WriteString("<div class=\"revisions-diff-frame\">\n")
	b.WriteString("<div class=\"revisions-diff\">\n")
	b.WriteString("<table class=\"diff\">\n")
	b.WriteString("<col class=\"content diffsplit left\" /><col class=\"content diffsplit middle\" /><col class=\"content diffsplit right\" />\n")
	b.WriteString("<tbody><tr><td colspan=\"3\"><p class=\"description\">GoPress revision compare stub (P7.33) — restore lands later.</p></td></tr></tbody>\n")
	b.WriteString("</table>\n</div>\n</div>\n")
	if revisionID != "" || from != "" || to != "" {
		b.WriteString("<p class=\"revisions-meta\">")
		b.WriteString("action=")
		b.WriteString(html.EscapeString(action))
		if revisionID != "" {
			b.WriteString(" revision=")
			b.WriteString(html.EscapeString(revisionID))
		}
		if from != "" {
			b.WriteString(" from=")
			b.WriteString(html.EscapeString(from))
		}
		if to != "" {
			b.WriteString(" to=")
			b.WriteString(html.EscapeString(to))
		}
		b.WriteString("</p>\n")
	}
	b.WriteString("</div>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// servePrivacyPolicyGuide is the Privacy Policy Guide screen stub (P8.7).
func (h *Handler) servePrivacyPolicyGuide(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Privacy Policy Guide &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui privacy-settings\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"privacy-settings-header\">\n")
	b.WriteString("<div class=\"privacy-settings-title-section\"><h1>Privacy</h1></div>\n")
	b.WriteString("<nav class=\"privacy-settings-tabs-wrapper\" aria-label=\"Secondary menu\">\n")
	b.WriteString("<a href=\"/wp-admin/options-privacy.php\" class=\"privacy-settings-tab\">Settings</a>\n")
	b.WriteString("<a href=\"/wp-admin/privacy-policy-guide.php\" class=\"privacy-settings-tab active\" aria-current=\"true\">Policy Guide</a>\n")
	b.WriteString("</nav>\n</div>\n")
	b.WriteString("<hr class=\"wp-header-end\">\n")
	b.WriteString("<div class=\"wrap privacy-settings-body\" id=\"privacy-settings\">\n")
	b.WriteString("<h2>Privacy Policy Guide</h2>\n")
	b.WriteString("<h3 class=\"section-title\">Introduction</h3>\n")
	b.WriteString("<p>This text template will help you to create your website&#8217;s privacy policy.</p>\n")
	b.WriteString("<p>The template contains a suggestion of sections you most likely will need. Under each section heading, you will find a short summary of what information you should provide.</p>\n")
	b.WriteString("<p>It is your responsibility to write a comprehensive privacy policy and to keep your policy current and accurate.</p>\n")
	b.WriteString("<div class=\"privacy-settings-accordion\">\n")
	b.WriteString("<h4 class=\"privacy-text-heading\">Who we are</h4>\n")
	b.WriteString("<div class=\"policy-text\"><p>In this section you should note your site URL, as well as the name of the company, organization, or individual behind this site.</p></div>\n")
	b.WriteString("<h4 class=\"privacy-text-heading\">Comments</h4>\n")
	b.WriteString("<div class=\"policy-text\"><p>In this section you should note what information is captured through comments.</p></div>\n")
	b.WriteString("<h4 class=\"privacy-text-heading\">Media</h4>\n")
	b.WriteString("<div class=\"policy-text\"><p>In this section you should note what information may be disclosed by uploaded images.</p></div>\n")
	b.WriteString("<h4 class=\"privacy-text-heading\">Cookies</h4>\n")
	b.WriteString("<div class=\"policy-text\"><p>In this section you should note which cookies your site uses.</p></div>\n")
	b.WriteString("</div>\n")
	b.WriteString("<p class=\"description\">GoPress privacy policy guide stub (P8.7).</p>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveCustomizerGone rejects the Customizer as out of scope (P7.6).
func (h *Handler) serveCustomizerGone(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Customizer &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Customizer unavailable</h1>\n")
	b.WriteString("<p>The Customizer is out of scope for GoPress. Use the <a href=\"/wp-admin/site-editor.php\">Site Editor</a> instead.</p>\n")
	b.WriteString("</body>\n</html>\n")
	writeHTML(w, http.StatusForbidden, b.String())
}

// serveThemeInstallGone rejects theme-install.php as out of scope (P7.18).
func (h *Handler) serveThemeInstallGone(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Add Themes &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Theme install unavailable</h1>\n")
	b.WriteString("<p>Installing themes from wordpress.org is out of scope for GoPress. Use the <a href=\"/wp-admin/themes.php\">Themes</a> screen to manage local themes.</p>\n")
	b.WriteString("</body>\n</html>\n")
	writeHTML(w, http.StatusForbidden, b.String())
}

// serveThemeEditorGone rejects theme-editor.php as out of scope (P7.37).
// PHP theme file editing is intentionally unavailable; use the Site Editor.
func (h *Handler) serveThemeEditorGone(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Edit Themes &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Theme editor unavailable</h1>\n")
	b.WriteString("<p>Editing PHP theme files via <code>theme-editor.php</code> is out of scope for GoPress. Use the <a href=\"/wp-admin/site-editor.php\">Site Editor</a> instead.</p>\n")
	b.WriteString("</body>\n</html>\n")
	writeHTML(w, http.StatusForbidden, b.String())
}

// servePluginInstallGone rejects plugin-install.php as out of scope (P7.19).
func (h *Handler) servePluginInstallGone(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Add Plugins &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Plugin install unavailable</h1>\n")
	b.WriteString("<p>Installing plugins from wordpress.org is out of scope for GoPress. Use the <a href=\"/wp-admin/plugins.php\">Plugins</a> screen to manage local plugins.</p>\n")
	b.WriteString("</body>\n</html>\n")
	writeHTML(w, http.StatusForbidden, b.String())
}

// servePluginEditorGone rejects plugin-editor.php as out of scope (P7.38).
// PHP plugin file editing is intentionally unavailable.
func (h *Handler) servePluginEditorGone(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Edit Plugins &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Plugin editor unavailable</h1>\n")
	b.WriteString("<p>Editing PHP plugin files via <code>plugin-editor.php</code> is out of scope for GoPress. Use the <a href=\"/wp-admin/plugins.php\">Plugins</a> screen to manage local plugins.</p>\n")
	b.WriteString("</body>\n</html>\n")
	writeHTML(w, http.StatusForbidden, b.String())
}

// serveUpdateCoreGone rejects update-core.php as out of scope (P7.20).
func (h *Handler) serveUpdateCoreGone(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>WordPress Updates &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Core updates unavailable</h1>\n")
	b.WriteString("<p>WordPress core updates are out of scope for GoPress. Deploy a new GoPress build instead of using <code>update-core.php</code>.</p>\n")
	b.WriteString("</body>\n</html>\n")
	writeHTML(w, http.StatusForbidden, b.String())
}

// serveLinkManagerGone rejects the legacy Links Manager as out of scope (P7.25).
func (h *Handler) serveLinkManagerGone(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Links &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Link Manager unavailable</h1>\n")
	b.WriteString("<p>The classic Links Manager (<code>link-manager.php</code>) is out of scope for GoPress.</p>\n")
	b.WriteString("</body>\n</html>\n")
	writeHTML(w, http.StatusForbidden, b.String())
}

func (h *Handler) serveUsers(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	users, err := listUsers(c, 20)
	if err != nil {
		http.Error(w, "users: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<h1 class=\"wp-heading-inline\">Users</h1>\n")
	b.WriteString("<a href=\"/wp-admin/user-new.php\" class=\"page-title-action\">Add New User</a>\n")
	b.WriteString("<hr class=\"wp-header-end\">\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped table-view-list users\">\n")
	b.WriteString("<thead><tr>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-username column-primary\">Username</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-name\">Name</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-email\">Email</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-role\">Role</th>")
	b.WriteString("<th scope=\"col\" class=\"manage-column column-posts\">Posts</th>")
	b.WriteString("</tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	for _, u := range users {
		b.WriteString("<tr id=\"user-")
		b.WriteString(strconv.FormatInt(u.ID, 10))
		b.WriteString("\">")
		b.WriteString("<td class=\"username column-username has-row-actions column-primary\" data-colname=\"Username\"><strong>")
		b.WriteString(html.EscapeString(u.Login))
		b.WriteString("</strong></td>")
		b.WriteString("<td class=\"name column-name\" data-colname=\"Name\">")
		b.WriteString(html.EscapeString(u.Name))
		b.WriteString("</td>")
		b.WriteString("<td class=\"email column-email\" data-colname=\"Email\"><a href=\"mailto:")
		b.WriteString(html.EscapeString(u.Email))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(u.Email))
		b.WriteString("</a></td>")
		b.WriteString("<td class=\"role column-role\" data-colname=\"Role\">")
		b.WriteString(html.EscapeString(u.Role))
		b.WriteString("</td>")
		b.WriteString("<td class=\"posts column-posts\" data-colname=\"Posts\">")
		b.WriteString(strconv.FormatInt(u.Posts, 10))
		b.WriteString("</td>")
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
	writeAdminPage(w, c, newAdminChrome(c, "Users", "users-php"), b.String())
}

// servePostEdit renders the classic post.php?action=edit stub shell (P7.24 / P7.32).
func (h *Handler) servePostEdit(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	action := r.URL.Query().Get("action")
	if action == "" {
		action = "edit"
	}
	if action != "edit" && action != "editattachment" {
		http.Error(w, "GoPress: post.php action not implemented\n", http.StatusNotImplemented)
		return
	}
	id, _ := strconv.ParseInt(r.URL.Query().Get("post"), 10, 64)
	if id <= 0 {
		http.Error(w, "Invalid post ID.\n", http.StatusBadRequest)
		return
	}
	post, err := wp.GetPostByID(c, id)
	if err != nil {
		http.Error(w, "post: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if post == nil {
		http.Error(w, "You attempted to edit an item that does not exist. Perhaps it was deleted?\n", http.StatusNotFound)
		return
	}
	if post.Type == "attachment" {
		h.serveAttachmentEdit(w, r, c, post)
		return
	}
	blogname := optionString(c, "blogname", "GoPress")
	heading := "Edit Post"
	if post.Type == "page" {
		heading = "Edit Page"
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(heading))
	b.WriteString(" &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n")
	b.WriteString("<script>var wpApiSettings = {\"root\":\"/wp-json/\",\"nonce\":\"")
	b.WriteString(html.EscapeString(c.CreateNonce("wp_rest")))
	b.WriteString("\"};</script>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui post-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1 class=\"wp-heading-inline\">")
	b.WriteString(html.EscapeString(heading))
	b.WriteString("</h1>\n")
	b.WriteString("<form name=\"post\" action=\"/wp-admin/post.php\" method=\"post\" id=\"post\">\n")
	b.WriteString("<input type=\"hidden\" id=\"post_ID\" name=\"post_ID\" value=\"")
	b.WriteString(strconv.FormatInt(post.ID, 10))
	b.WriteString("\" />\n")
	b.WriteString("<input type=\"hidden\" id=\"post_type\" name=\"post_type\" value=\"")
	b.WriteString(html.EscapeString(post.Type))
	b.WriteString("\" />\n")
	b.WriteString("<input type=\"hidden\" id=\"original_post_status\" name=\"original_post_status\" value=\"")
	b.WriteString(html.EscapeString(post.Status))
	b.WriteString("\" />\n")
	b.WriteString("<div id=\"poststuff\">\n")
	b.WriteString("<div id=\"post-body\" class=\"metabox-holder columns-2\">\n")
	b.WriteString("<div id=\"post-body-content\">\n")
	b.WriteString("<div id=\"titlediv\"><div id=\"titlewrap\">\n")
	b.WriteString("<label class=\"screen-reader-text\" id=\"title-prompt-text\" for=\"title\">Add title</label>\n")
	b.WriteString("<input type=\"text\" name=\"post_title\" size=\"30\" value=\"")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString("\" id=\"title\" spellcheck=\"true\" autocomplete=\"off\" />\n")
	b.WriteString("</div></div>\n")
	b.WriteString("<div id=\"postdivrich\" class=\"postarea wp-editor-expand\">\n")
	b.WriteString("<div id=\"wp-content-editor-container\" class=\"wp-editor-container\">\n")
	b.WriteString("<textarea class=\"wp-editor-area\" style=\"height:300px\" autocomplete=\"off\" cols=\"40\" name=\"content\" id=\"content\">")
	b.WriteString(html.EscapeString(post.Content))
	b.WriteString("</textarea>\n")
	b.WriteString("</div>\n")
	b.WriteString("<p class=\"description\">GoPress post editor stub (P7.24) — save/publish lands later.</p>\n")
	b.WriteString("</div>\n</div>\n")
	b.WriteString("<div id=\"postbox-container-1\" class=\"postbox-container\">\n")
	b.WriteString("<div id=\"submitdiv\" class=\"postbox\">\n")
	b.WriteString("<div class=\"postbox-header\"><h2 class=\"hndle\">Publish</h2></div>\n")
	b.WriteString("<div class=\"inside\">\n")
	b.WriteString("<div class=\"submitbox\" id=\"submitpost\">\n")
	b.WriteString("<div id=\"minor-publishing\"><div id=\"misc-publishing-actions\">\n")
	b.WriteString("<div class=\"misc-pub-section misc-pub-post-status\">Status: <span id=\"post-status-display\"><strong>")
	b.WriteString(html.EscapeString(displayPostStatus(post.Status)))
	b.WriteString("</strong></span></div>\n")
	b.WriteString("</div></div>\n")
	b.WriteString("<div id=\"major-publishing-actions\">\n")
	b.WriteString("<div id=\"publishing-action\"><input type=\"submit\" name=\"publish\" id=\"publish\" class=\"button button-primary button-large\" value=\"Update\" /></div>\n")
	b.WriteString("<div class=\"clear\"></div>\n</div>\n</div>\n</div>\n</div>\n</div>\n")
	b.WriteString("</div>\n</div>\n</div>\n</form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveAttachmentEdit is the attachment / media edit screen stub (P7.32).
func (h *Handler) serveAttachmentEdit(w http.ResponseWriter, r *http.Request, c *wp.Context, post *wp.Post) {
	if post == nil {
		http.Error(w, "Invalid post ID.\n", http.StatusBadRequest)
		return
	}
	alt := ""
	if c != nil && c.Meta != nil {
		alt = asString(c.Meta.GetMetadata(wp.MetaPost, post.ID, "_wp_attachment_image_alt", true))
	}
	url := wp.GetAttachmentURL(c, post.ID)
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Edit Media &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui post-php post-type-attachment\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1 class=\"wp-heading-inline\">Edit Media</h1>\n")
	b.WriteString("<a href=\"/wp-admin/upload.php\" class=\"page-title-action\">Media Library</a>\n")
	b.WriteString("<form name=\"post\" action=\"/wp-admin/post.php?post=")
	b.WriteString(strconv.FormatInt(post.ID, 10))
	b.WriteString("&amp;action=edit\" method=\"post\" id=\"post\">\n")
	b.WriteString("<input type=\"hidden\" id=\"post_ID\" name=\"post_ID\" value=\"")
	b.WriteString(strconv.FormatInt(post.ID, 10))
	b.WriteString("\" />\n")
	b.WriteString("<input type=\"hidden\" id=\"post_type\" name=\"post_type\" value=\"attachment\" />\n")
	b.WriteString("<input type=\"hidden\" name=\"action\" value=\"editattachment\" />\n")
	b.WriteString("<div id=\"poststuff\">\n<div id=\"post-body\" class=\"metabox-holder columns-2\">\n")
	b.WriteString("<div id=\"post-body-content\">\n")
	if url != "" {
		b.WriteString("<div class=\"wp_attachment_holder\"><p class=\"attachment-url\"><a href=\"")
		b.WriteString(html.EscapeString(url))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(url))
		b.WriteString("</a></p></div>\n")
	}
	b.WriteString("<div id=\"titlediv\"><div id=\"titlewrap\">\n")
	b.WriteString("<label class=\"screen-reader-text\" for=\"title\">Title</label>\n")
	b.WriteString("<input type=\"text\" name=\"post_title\" size=\"30\" value=\"")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString("\" id=\"title\" spellcheck=\"true\" autocomplete=\"off\" />\n")
	b.WriteString("</div></div>\n")
	b.WriteString("<div class=\"attachment-alt\">\n")
	b.WriteString("<p><label for=\"_wp_attachment_image_alt\">Alternative Text</label><br />\n")
	b.WriteString("<input type=\"text\" name=\"_wp_attachment_image_alt\" id=\"_wp_attachment_image_alt\" value=\"")
	b.WriteString(html.EscapeString(alt))
	b.WriteString("\" class=\"widefat\" /></p>\n")
	b.WriteString("<p><label for=\"attachment_caption\">Caption</label><br />\n")
	b.WriteString("<textarea name=\"excerpt\" id=\"attachment_caption\" rows=\"3\" cols=\"40\">")
	b.WriteString(html.EscapeString(post.Excerpt))
	b.WriteString("</textarea></p>\n")
	b.WriteString("<p><label for=\"attachment_content\">Description</label><br />\n")
	b.WriteString("<textarea name=\"content\" id=\"content\" rows=\"5\" cols=\"40\">")
	b.WriteString(html.EscapeString(post.Content))
	b.WriteString("</textarea></p>\n")
	b.WriteString("</div>\n")
	b.WriteString("<p class=\"description\">GoPress attachment editor stub (P7.32).</p>\n")
	b.WriteString("</div>\n")
	b.WriteString("<div id=\"postbox-container-1\" class=\"postbox-container\">\n")
	b.WriteString("<div id=\"submitdiv\" class=\"postbox\">\n")
	b.WriteString("<div class=\"postbox-header\"><h2 class=\"hndle\">Save</h2></div>\n")
	b.WriteString("<div class=\"inside\"><div class=\"submitbox\" id=\"submitpost\">\n")
	b.WriteString("<div id=\"major-publishing-actions\">\n")
	b.WriteString("<div id=\"publishing-action\"><input type=\"submit\" name=\"save\" id=\"publish\" class=\"button button-primary button-large\" value=\"Update\" /></div>\n")
	b.WriteString("<div class=\"clear\"></div>\n</div></div></div></div></div>\n")
	b.WriteString("</div></div></form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

// serveUserNew renders the Add New User form stub (P7.21). POST create is not implemented.
func (h *Handler) serveUserNew(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Add New User &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<h1>Add New User</h1>\n")
	b.WriteString("<p>Create a brand new user and add them to this site.</p>\n")
	b.WriteString("<form method=\"post\" name=\"createuser\" id=\"createuser\" class=\"validate\" novalidate=\"novalidate\">\n")
	b.WriteString("<input name=\"action\" type=\"hidden\" value=\"createuser\" />\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr class=\"form-field form-required\"><th scope=\"row\"><label for=\"user_login\">Username <span class=\"description\">(required)</span></label></th>")
	b.WriteString("<td><input name=\"user_login\" type=\"text\" id=\"user_login\" class=\"ltr\" value=\"\" aria-required=\"true\" maxlength=\"60\" /></td></tr>\n")
	b.WriteString("<tr class=\"form-field form-required\"><th scope=\"row\"><label for=\"email\">Email <span class=\"description\">(required)</span></label></th>")
	b.WriteString("<td><input name=\"email\" type=\"email\" id=\"email\" value=\"\" /></td></tr>\n")
	b.WriteString("<tr class=\"form-field\"><th scope=\"row\"><label for=\"first_name\">First Name</label></th>")
	b.WriteString("<td><input name=\"first_name\" type=\"text\" id=\"first_name\" value=\"\" /></td></tr>\n")
	b.WriteString("<tr class=\"form-field\"><th scope=\"row\"><label for=\"last_name\">Last Name</label></th>")
	b.WriteString("<td><input name=\"last_name\" type=\"text\" id=\"last_name\" value=\"\" /></td></tr>\n")
	b.WriteString("<tr class=\"form-field\"><th scope=\"row\"><label for=\"url\">Website</label></th>")
	b.WriteString("<td><input name=\"url\" type=\"url\" id=\"url\" class=\"code\" value=\"\" /></td></tr>\n")
	b.WriteString("<tr class=\"form-field\"><th scope=\"row\"><label for=\"role\">Role</label></th>")
	b.WriteString("<td><select name=\"role\" id=\"role\">")
	b.WriteString("<option selected=\"selected\" value=\"subscriber\">Subscriber</option>")
	b.WriteString("<option value=\"contributor\">Contributor</option>")
	b.WriteString("<option value=\"author\">Author</option>")
	b.WriteString("<option value=\"editor\">Editor</option>")
	b.WriteString("<option value=\"administrator\">Administrator</option>")
	b.WriteString("</select></td></tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"createuser\" id=\"createusersub\" class=\"button button-primary\" value=\"Add New User\" /></p>\n")
	b.WriteString("</form>\n")
	b.WriteString("<p class=\"description\">User creation via this form is not yet implemented in GoPress.</p>\n")
	b.WriteString("</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

type postRow struct {
	ID     int64
	Title  string
	Date   string
	Status string
	Author string
}

func countPostsByTypeStatus(c *wp.Context, postType, status string) int {
	if c == nil || c.DB == nil {
		return 0
	}
	q, args, err := c.DB.Prepare(
		"SELECT COUNT(*) AS n FROM "+c.DB.Posts()+" WHERE post_type = %s AND post_status = %s",
		postType, status,
	)
	if err != nil {
		return 0
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return 0
	}
	return int(asInt64(row["n"]))
}

func countComments(c *wp.Context, approved string) int {
	if c == nil || c.DB == nil {
		return 0
	}
	q, args, err := c.DB.Prepare(
		"SELECT COUNT(*) AS n FROM "+c.DB.Comments()+" WHERE comment_approved = %s",
		approved,
	)
	if err != nil {
		return 0
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return 0
	}
	return int(asInt64(row["n"]))
}

func recentPostsByType(c *wp.Context, postType string, limit int) ([]postRow, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 20
	}
	if postType == "" {
		postType = "post"
	}
	q, args, err := c.DB.Prepare(
		"SELECT p.ID, p.post_title, p.post_date, p.post_status, p.post_author, u.display_name, u.user_login FROM "+
			c.DB.Posts()+" p LEFT JOIN "+c.DB.Users()+" u ON p.post_author = u.ID "+
			" WHERE p.post_type = %s AND p.post_status IN (%s, %s, %s, %s, %s) "+
			"ORDER BY p.post_date DESC LIMIT %d",
		postType, "publish", "draft", "pending", "private", "future", limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]postRow, 0, len(rows))
	for _, row := range rows {
		author := asString(row["display_name"])
		if author == "" {
			author = asString(row["user_login"])
		}
		if author == "" {
			author = "—"
		}
		title := asString(row["post_title"])
		if strings.TrimSpace(title) == "" {
			title = "(no title)"
		}
		out = append(out, postRow{
			ID:     asInt64(row["ID"]),
			Title:  title,
			Date:   asString(row["post_date"]),
			Status: asString(row["post_status"]),
			Author: author,
		})
	}
	return out, nil
}

type mediaRow struct {
	ID    int64
	Title string
	Mime  string
}

type commentRow struct {
	ID      int64
	Author  string
	Content string
}

type userRow struct {
	ID    int64
	Login string
	Name  string
	Email string
	Role  string
	Posts int64
}

func recentMedia(c *wp.Context, limit int) ([]mediaRow, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 20
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_title, post_mime_type FROM "+c.DB.Posts()+
			" WHERE post_type = %s ORDER BY post_date DESC LIMIT %d",
		"attachment", limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]mediaRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mediaRow{
			ID:    asInt64(row["ID"]),
			Title: asString(row["post_title"]),
			Mime:  asString(row["post_mime_type"]),
		})
	}
	return out, nil
}

func recentComments(c *wp.Context, limit int) ([]commentRow, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 20
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID, comment_author, comment_content FROM "+c.DB.Comments()+
			" ORDER BY comment_date DESC LIMIT %d",
		limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]commentRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, commentRow{
			ID:      asInt64(row["comment_ID"]),
			Author:  asString(row["comment_author"]),
			Content: asString(row["comment_content"]),
		})
	}
	return out, nil
}

func listUsers(c *wp.Context, limit int) ([]userRow, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 20
	}
	q, args, err := c.DB.Prepare(
		"SELECT u.ID, u.user_login, u.display_name, u.user_email, "+
			"(SELECT COUNT(*) FROM "+c.DB.Posts()+" p WHERE p.post_author = u.ID AND p.post_type = 'post' AND p.post_status <> 'trash') AS post_count "+
			"FROM "+c.DB.Users()+" u ORDER BY u.user_login ASC LIMIT %d",
		limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]userRow, 0, len(rows))
	for _, row := range rows {
		id := asInt64(row["ID"])
		u := userRow{
			ID:    id,
			Login: asString(row["user_login"]),
			Name:  asString(row["display_name"]),
			Email: asString(row["user_email"]),
			Posts: asInt64(row["post_count"]),
			Role:  "—",
		}
		if c.Roles != nil {
			user, err := c.GetUserBy("id", id)
			if err == nil && user != nil {
				_ = c.Roles.InitUser(user, c.Meta)
				if len(user.Roles) > 0 {
					u.Role = strings.ReplaceAll(user.Roles[0], "_", " ")
					if len(u.Role) > 0 {
						u.Role = strings.ToUpper(u.Role[:1]) + u.Role[1:]
					}
				}
			}
		}
		out = append(out, u)
	}
	return out, nil
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func adminNoticesHTML(c *wp.Context) string {
	if c == nil {
		return ""
	}
	return c.CaptureAdminNotices()
}

func optionString(c *wp.Context, name, fallback string) string {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case string:
		if t == "" {
			return fallback
		}
		return t
	default:
		s := fmt.Sprint(t)
		if s == "" || s == "<nil>" {
			return fallback
		}
		return s
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case uint64:
		return int64(t)
	case []byte:
		n, _ := strconv.ParseInt(string(t), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return n
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}
