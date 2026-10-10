// Package admin implements minimal wp-admin screen stubs (P7.1+).
package admin

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	case "/wp-admin/options-general.php":
		h.requireLogin(w, r, c, h.serveOptionsGeneral)
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
	case "/wp-admin/edit-comments.php":
		h.requireLogin(w, r, c, h.serveEditComments)
		return true
	case "/wp-admin/themes.php":
		h.requireLogin(w, r, c, h.serveThemes)
		return true
	case "/wp-admin/users.php":
		h.requireLogin(w, r, c, h.serveUsers)
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
	case "/wp-admin/export.php":
		h.requireLogin(w, r, c, h.serveExport)
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
	writeHTML(w, http.StatusOK, fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>Dashboard &lsaquo; %s &#8212; WordPress</title>
<script>var wpApiSettings = {"root":"/wp-json/","nonce":"%s"};</script>
<script>var heartbeatSettings = {"nonce":"%s"};</script>
</head>
<body class="wp-admin">
%s<h1>Dashboard</h1>
<p>%s</p>
<div id="dashboard-widgets-wrap">
<div id="dashboard-widgets" class="metabox-holder">
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
</div>
</div>
</body>
</html>
`, html.EscapeString(blogname), html.EscapeString(restNonce), html.EscapeString(hbNonce), adminNoticesHTML(c), html.EscapeString(blogname), posts, pages, comments))
}

func (h *Handler) serveEditPosts(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := recentPosts(c, 20)
	if err != nil {
		http.Error(w, "posts: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Posts &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString("<h1>Posts</h1>\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped table-view-list posts\">\n")
	b.WriteString("<thead><tr><th scope=\"col\" class=\"column-title\">Title</th><th scope=\"col\" class=\"column-date\">Date</th></tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	for _, p := range posts {
		b.WriteString("<tr id=\"post-")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\"><td class=\"title column-title\"><strong>")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</strong></td><td class=\"date column-date\">")
		b.WriteString(html.EscapeString(p.Date))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) serveOptionsGeneral(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>General Settings &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>General Settings</h1>\n")
	b.WriteString("<form method=\"post\" action=\"options.php\">\n")
	b.WriteString("<table class=\"form-table\" role=\"presentation\">\n")
	b.WriteString("<tr>\n<th scope=\"row\"><label for=\"blogname\">Site Title</label></th>\n")
	b.WriteString("<td><input name=\"blogname\" type=\"text\" id=\"blogname\" value=\"")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("\" class=\"regular-text\"></td>\n</tr>\n")
	b.WriteString("</table>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Save Changes\"></p>\n")
	b.WriteString("</form>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) servePlugins(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Plugins &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Plugins</h1>\n")
	b.WriteString("<table class=\"wp-list-table widefat plugins\">\n")
	b.WriteString("<thead><tr><th scope=\"col\">Plugin</th><th scope=\"col\">Description</th></tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	b.WriteString("<tr class=\"inactive\"><td class=\"plugin-title\"><strong>No plugins installed</strong></td>")
	b.WriteString("<td class=\"column-description desc\"><p>GoPress plugin management stub (P7.1).</p></td></tr>\n")
	b.WriteString("</tbody>\n</table>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) serveProfile(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	user := c.CurrentUser
	login := ""
	email := ""
	display := ""
	if user != nil {
		login = user.Login
		email = user.Email
		display = user.DisplayName
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Profile &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString("<h1>Profile</h1>\n")
	b.WriteString("<form id=\"your-profile\">\n")
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
	b.WriteString("</table>\n</form>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) serveUpload(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	items, err := recentMedia(c, 20)
	if err != nil {
		http.Error(w, "media: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Media Library &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<h1>Media Library</h1>\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped media\">\n")
	b.WriteString("<thead><tr><th scope=\"col\" class=\"column-title\">File</th><th scope=\"col\" class=\"column-mime\">Type</th></tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	if len(items) == 0 {
		b.WriteString("<tr class=\"no-items\"><td class=\"colspanchange\" colspan=\"2\">No media items found.</td></tr>\n")
	}
	for _, m := range items {
		b.WriteString("<tr id=\"post-")
		b.WriteString(strconv.FormatInt(m.ID, 10))
		b.WriteString("\"><td class=\"title column-title\"><strong>")
		b.WriteString(html.EscapeString(m.Title))
		b.WriteString("</strong></td><td class=\"column-mime\">")
		b.WriteString(html.EscapeString(m.Mime))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
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
	blogname := optionString(c, "blogname", "GoPress")
	items, err := recentComments(c, 20)
	if err != nil {
		http.Error(w, "comments: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Comments &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<h1>Comments</h1>\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped comments\">\n")
	b.WriteString("<thead><tr><th scope=\"col\" class=\"column-author\">Author</th><th scope=\"col\" class=\"column-comment\">Comment</th></tr></thead>\n")
	b.WriteString("<tbody id=\"the-comment-list\">\n")
	if len(items) == 0 {
		b.WriteString("<tr class=\"no-items\"><td class=\"colspanchange\" colspan=\"2\">No comments found.</td></tr>\n")
	}
	for _, item := range items {
		b.WriteString("<tr id=\"comment-")
		b.WriteString(strconv.FormatInt(item.ID, 10))
		b.WriteString("\"><td class=\"author column-author\">")
		b.WriteString(html.EscapeString(item.Author))
		b.WriteString("</td><td class=\"comment column-comment\"><p>")
		b.WriteString(html.EscapeString(item.Content))
		b.WriteString("</p></td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) serveThemes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	stylesheet := optionString(c, "stylesheet", "twentytwentytwo")
	template := optionString(c, "template", stylesheet)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Themes &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<h1>Themes</h1>\n")
	b.WriteString("<div class=\"theme-browser\">\n")
	b.WriteString("<div class=\"theme active\" data-slug=\"")
	b.WriteString(html.EscapeString(stylesheet))
	b.WriteString("\">\n")
	b.WriteString("<h2 class=\"theme-name\">")
	b.WriteString(html.EscapeString(stylesheet))
	b.WriteString(" <span>Active</span></h2>\n")
	b.WriteString("<p class=\"theme-template\">Template: ")
	b.WriteString(html.EscapeString(template))
	b.WriteString("</p>\n")
	b.WriteString("<p>GoPress themes stub (P7.4).</p>\n")
	b.WriteString("</div>\n</div>\n</body>\n</html>\n")
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

// serveNavMenus is a classic menus admin stub (P7.9).
func (h *Handler) serveNavMenus(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	menus, err := listNavMenus(c)
	if err != nil {
		http.Error(w, "menus: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Menus &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui nav-menus-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\" id=\"menu-management\">\n")
	b.WriteString("<h1>Menus</h1>\n")
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
	b.WriteString("</ul>\n</div>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
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

// serveExportPersonalData is a privacy tools stub (P8.2).
func (h *Handler) serveExportPersonalData(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	h.servePrivacyStub(w, c, "Export Personal Data",
		"Manage requests for an export of personal data.",
		"GoPress privacy export stub — full request workflow lands later.")
}

// serveErasePersonalData is a privacy tools stub (P8.2).
func (h *Handler) serveErasePersonalData(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	h.servePrivacyStub(w, c, "Erase Personal Data",
		"Manage requests to erase personal data.",
		"GoPress privacy erasure stub — full request workflow lands later.")
}

// serveOptionsPrivacy is a privacy settings stub (P8.2).
func (h *Handler) serveOptionsPrivacy(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	h.servePrivacyStub(w, c, "Privacy",
		"Privacy settings and policy page.",
		"GoPress privacy settings stub — policy guide lands later.")
}

func (h *Handler) servePrivacyStub(w http.ResponseWriter, c *wp.Context, title, lead, note string) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString(" &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui privacy-settings\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\" id=\"privacy-settings\">\n")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>\n")
	b.WriteString("<p>")
	b.WriteString(html.EscapeString(lead))
	b.WriteString("</p>\n")
	b.WriteString("<p class=\"description\">")
	b.WriteString(html.EscapeString(note))
	b.WriteString("</p>\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped privacy-requests\">\n")
	b.WriteString("<thead><tr><th scope=\"col\">Requester</th><th scope=\"col\">Status</th></tr></thead>\n")
	b.WriteString("<tbody><tr class=\"no-items\"><td class=\"colspanchange\" colspan=\"2\">No requests found.</td></tr></tbody>\n")
	b.WriteString("</table>\n</div>\n</body>\n</html>\n")
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

func (h *Handler) serveUsers(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	users, err := listUsers(c, 20)
	if err != nil {
		http.Error(w, "users: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Users &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	b.WriteString("<body class=\"wp-admin wp-core-ui\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<h1>Users</h1>\n")
	b.WriteString("<table class=\"wp-list-table widefat fixed striped users\">\n")
	b.WriteString("<thead><tr><th scope=\"col\" class=\"column-username\">Username</th><th scope=\"col\" class=\"column-email\">Email</th></tr></thead>\n")
	b.WriteString("<tbody id=\"the-list\">\n")
	for _, u := range users {
		b.WriteString("<tr id=\"user-")
		b.WriteString(strconv.FormatInt(u.ID, 10))
		b.WriteString("\"><td class=\"username column-username\"><strong>")
		b.WriteString(html.EscapeString(u.Login))
		b.WriteString("</strong></td><td class=\"email column-email\">")
		b.WriteString(html.EscapeString(u.Email))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

type postRow struct {
	ID    int64
	Title string
	Date  string
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

func recentPosts(c *wp.Context, limit int) ([]postRow, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 20
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
	out := make([]postRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, postRow{
			ID:    asInt64(row["ID"]),
			Title: asString(row["post_title"]),
			Date:  asString(row["post_date"]),
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
	Email string
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
		"SELECT ID, user_login, user_email FROM "+c.DB.Users()+
			" ORDER BY user_login ASC LIMIT %d",
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
		out = append(out, userRow{
			ID:    asInt64(row["ID"]),
			Login: asString(row["user_login"]),
			Email: asString(row["user_email"]),
		})
	}
	return out, nil
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
