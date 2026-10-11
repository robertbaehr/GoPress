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

const wxrVersion = "1.2"

// serveExport is the Tools → Export screen (P7.10 / P7.53 / P7.90).
func (h *Handler) serveExport(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P7.90: export capability gate (mirrors export.php wp_die).
	if c == nil || c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "export") {
		http.Error(w, "Sorry, you are not allowed to export the content of this site.\n", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodGet && (r.URL.Query().Get("download") == "true" || r.URL.Query().Get("download") == "1") {
		h.serveWXRDownload(w, r, c)
		return
	}
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Export &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	// P7.90: export-php body class like get_body_class for this screen.
	b.WriteString("<body class=\"wp-admin wp-core-ui export-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Export</h1>\n")
	b.WriteString("<p>When you click the button below WordPress will create an XML file for you to save to your computer.</p>\n")
	b.WriteString("<p>This format, which is called WordPress eXtended RSS or WXR, will contain your posts, pages, comments, custom fields, categories, and tags.</p>\n")
	b.WriteString("<p>Once you&#8217;ve saved the download file, you can use the Import function in another WordPress installation to import the content from this site.</p>\n")
	b.WriteString("<h2>Choose what to export</h2>\n")
	b.WriteString("<form method=\"get\" id=\"export-filters\" action=\"/wp-admin/export.php\">\n")
	b.WriteString("<input type=\"hidden\" name=\"download\" value=\"true\">\n")
	b.WriteString("<fieldset><legend class=\"screen-reader-text\">Content to export</legend>\n")
	b.WriteString("<p><label><input type=\"radio\" name=\"content\" value=\"all\" checked=\"checked\" aria-describedby=\"all-content-desc\"> All content</label></p>\n")
	b.WriteString("<p class=\"description\" id=\"all-content-desc\">This will contain all of your posts, pages, comments, custom fields, terms, navigation menus, and custom posts.</p>\n")
	b.WriteString("<p><label><input type=\"radio\" name=\"content\" value=\"posts\"> Posts</label></p>\n")
	b.WriteString("<ul id=\"post-filters\" class=\"export-filters\">\n")
	b.WriteString("<li><label for=\"post-status\" class=\"label-responsive\">Status:</label> ")
	b.WriteString(`<select name="post_status" id="post-status"><option value="0">All</option>`)
	b.WriteString(`<option value="publish">Published</option><option value="draft">Draft</option>`)
	b.WriteString(`<option value="pending">Pending</option><option value="private">Private</option></select></li>` + "\n")
	b.WriteString("<li><label for=\"post-start-date\" class=\"label-responsive\">Start date:</label> ")
	b.WriteString(`<select name="post_start_date" id="post-start-date"><option value="0">&mdash; Select &mdash;</option></select> `)
	b.WriteString(`<label for="post-end-date" class="label-responsive">End date:</label> `)
	b.WriteString(`<select name="post_end_date" id="post-end-date"><option value="0">&mdash; Select &mdash;</option></select></li>` + "\n")
	b.WriteString("</ul>\n")
	b.WriteString("<p><label><input type=\"radio\" name=\"content\" value=\"pages\"> Pages</label></p>\n")
	b.WriteString("<ul id=\"page-filters\" class=\"export-filters\">\n")
	b.WriteString("<li><label for=\"page-status\" class=\"label-responsive\">Status:</label> ")
	b.WriteString(`<select name="page_status" id="page-status"><option value="0">All</option>`)
	b.WriteString(`<option value="publish">Published</option><option value="draft">Draft</option></select></li>` + "\n")
	b.WriteString("</ul>\n")
	b.WriteString("<p><label><input type=\"radio\" name=\"content\" value=\"attachment\"> Media</label></p>\n")
	b.WriteString("<ul id=\"attachment-filters\" class=\"export-filters\">\n")
	b.WriteString("<li><span class=\"description\">Export attached media files as WXR items.</span></li>\n")
	b.WriteString("</ul>\n")
	b.WriteString("</fieldset>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Download Export File\"></p>\n")
	b.WriteString("</form>\n</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) serveWXRDownload(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	args := exportArgsFromRequest(r)
	// P7.90: export_args filter (mirrors apply_filters('export_args', $args)).
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("export_args", args); v != nil {
			if m, ok := v.(map[string]string); ok {
				args = m
			}
		}
	}
	content := args["content"]
	if content == "" {
		content = "all"
	}
	xmlBody, err := buildMinimalWXR(c, content, args)
	if err != nil {
		http.Error(w, "Export failed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	sitename := sanitizeExportKey(optionString(c, "blogname", "gopress"))
	filename := fmt.Sprintf("%s.WordPress.%s.xml", sitename, time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xmlBody))
}

func exportArgsFromRequest(r *http.Request) map[string]string {
	q := r.URL.Query()
	content := strings.TrimSpace(q.Get("content"))
	if content == "" {
		content = "all"
	}
	args := map[string]string{"content": content}
	switch content {
	case "posts", "post":
		args["content"] = "post"
		if cat := strings.TrimSpace(q.Get("cat")); cat != "" && cat != "0" {
			args["category"] = cat
		}
		if a := strings.TrimSpace(q.Get("post_author")); a != "" && a != "0" {
			args["author"] = a
		}
		if s := strings.TrimSpace(q.Get("post_start_date")); s != "" && s != "0" {
			args["start_date"] = s
		}
		if e := strings.TrimSpace(q.Get("post_end_date")); e != "" && e != "0" {
			args["end_date"] = e
		}
		if st := strings.TrimSpace(q.Get("post_status")); st != "" && st != "0" {
			args["status"] = st
		}
	case "pages", "page":
		args["content"] = "page"
		if a := strings.TrimSpace(q.Get("page_author")); a != "" && a != "0" {
			args["author"] = a
		}
		if s := strings.TrimSpace(q.Get("page_start_date")); s != "" && s != "0" {
			args["start_date"] = s
		}
		if e := strings.TrimSpace(q.Get("page_end_date")); e != "" && e != "0" {
			args["end_date"] = e
		}
		if st := strings.TrimSpace(q.Get("page_status")); st != "" && st != "0" {
			args["status"] = st
		}
	case "attachment", "media":
		args["content"] = "attachment"
		if s := strings.TrimSpace(q.Get("attachment_start_date")); s != "" && s != "0" {
			args["start_date"] = s
		}
		if e := strings.TrimSpace(q.Get("attachment_end_date")); e != "" && e != "0" {
			args["end_date"] = e
		}
	default:
		args["content"] = "all"
	}
	return args
}

func buildMinimalWXR(c *wp.Context, content string, args map[string]string) (string, error) {
	if c == nil || c.DB == nil {
		return "", fmt.Errorf("nil context")
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	blogname := optionString(c, "blogname", "GoPress")
	desc := optionString(c, "blogdescription", "")
	lang := optionString(c, "WPLANG", "en-US")
	if lang == "" {
		lang = "en-US"
	}

	types := contentTypesForExport(content)
	status := "publish"
	if args != nil {
		if st := strings.TrimSpace(args["status"]); st != "" && st != "0" {
			status = st
		}
	}
	var authorID int64
	if args != nil {
		if a := strings.TrimSpace(args["author"]); a != "" {
			if n, err := strconv.ParseInt(a, 10, 64); err == nil && n > 0 {
				authorID = n
			}
		}
	}

	var items []wp.Post
	for _, pt := range types {
		qArgs := wp.QueryArgs{
			PostType:     pt,
			PostStatus:   status,
			PostsPerPage: -1,
		}
		if authorID > 0 {
			qArgs.Author = authorID
		}
		// Attachments use inherit status in WP; keep publish query for posts/pages
		// and fall back to inherit for attachment when status is default publish.
		if pt == "attachment" && (status == "publish" || status == "") {
			qArgs.PostStatus = "inherit"
		}
		rows, err := wp.GetPosts(c, qArgs)
		if err != nil {
			return "", err
		}
		items = append(items, rows...)
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" ?>` + "\n")
	b.WriteString(wp.TheGenerator(c, "export"))
	b.WriteString("\n")
	b.WriteString("<!-- This is a WordPress eXtended RSS (WXR) export file generated by GoPress. -->\n")
	b.WriteString(`<rss version="2.0"` + "\n")
	b.WriteString(`	xmlns:excerpt="http://wordpress.org/export/` + wxrVersion + `/excerpt/"` + "\n")
	b.WriteString(`	xmlns:content="http://purl.org/rss/1.0/modules/content/"` + "\n")
	b.WriteString(`	xmlns:wfw="http://wellformedweb.org/CommentAPI/"` + "\n")
	b.WriteString(`	xmlns:dc="http://purl.org/dc/elements/1.1/"` + "\n")
	b.WriteString(`	xmlns:wp="http://wordpress.org/export/` + wxrVersion + `/"` + "\n")
	b.WriteString(">\n\n<channel>\n")
	b.WriteString("	<title>" + wxrCDATA(blogname) + "</title>\n")
	b.WriteString("	<link>" + html.EscapeString(home) + "</link>\n")
	b.WriteString("	<description>" + wxrCDATA(desc) + "</description>\n")
	b.WriteString("	<pubDate>" + time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 +0000") + "</pubDate>\n")
	b.WriteString("	<language>" + html.EscapeString(lang) + "</language>\n")
	b.WriteString("	<wp:wxr_version>" + wxrVersion + "</wp:wxr_version>\n")
	b.WriteString("	<wp:base_site_url>" + html.EscapeString(home) + "</wp:base_site_url>\n")
	b.WriteString("	<wp:base_blog_url>" + html.EscapeString(home) + "</wp:base_blog_url>\n")

	for _, post := range items {
		link := home + "/?p=" + fmt.Sprintf("%d", post.ID)
		creator := ""
		if u, err := c.GetUserBy("id", post.Author); err == nil && u != nil {
			creator = u.Login
		}
		b.WriteString("	<item>\n")
		b.WriteString("		<title>" + wxrCDATA(post.Title) + "</title>\n")
		b.WriteString("		<link>" + html.EscapeString(link) + "</link>\n")
		b.WriteString("		<pubDate>" + mysqlDateToRFC822(post.DateGMT) + "</pubDate>\n")
		b.WriteString("		<dc:creator>" + wxrCDATA(creator) + "</dc:creator>\n")
		b.WriteString(`		<guid isPermaLink="false">` + html.EscapeString(post.GUID) + "</guid>\n")
		b.WriteString("		<description></description>\n")
		b.WriteString("		<content:encoded>" + wxrCDATA(post.Content) + "</content:encoded>\n")
		b.WriteString("		<excerpt:encoded>" + wxrCDATA(post.Excerpt) + "</excerpt:encoded>\n")
		b.WriteString("		<wp:post_id>" + fmt.Sprintf("%d", post.ID) + "</wp:post_id>\n")
		b.WriteString("		<wp:post_date>" + wxrCDATA(post.Date) + "</wp:post_date>\n")
		b.WriteString("		<wp:post_date_gmt>" + wxrCDATA(post.DateGMT) + "</wp:post_date_gmt>\n")
		b.WriteString("		<wp:post_modified>" + wxrCDATA(post.Modified) + "</wp:post_modified>\n")
		b.WriteString("		<wp:post_modified_gmt>" + wxrCDATA(post.ModifiedGMT) + "</wp:post_modified_gmt>\n")
		b.WriteString("		<wp:comment_status>" + wxrCDATA("open") + "</wp:comment_status>\n")
		b.WriteString("		<wp:ping_status>" + wxrCDATA("closed") + "</wp:ping_status>\n")
		b.WriteString("		<wp:post_name>" + wxrCDATA(post.Name) + "</wp:post_name>\n")
		b.WriteString("		<wp:status>" + wxrCDATA(post.Status) + "</wp:status>\n")
		b.WriteString("		<wp:post_parent>" + fmt.Sprintf("%d", post.Parent) + "</wp:post_parent>\n")
		b.WriteString("		<wp:menu_order>0</wp:menu_order>\n")
		b.WriteString("		<wp:post_type>" + wxrCDATA(post.Type) + "</wp:post_type>\n")
		b.WriteString("		<wp:post_password>" + wxrCDATA(post.Password) + "</wp:post_password>\n")
		b.WriteString("		<wp:is_sticky>0</wp:is_sticky>\n")
		b.WriteString("	</item>\n")
	}

	b.WriteString("</channel>\n</rss>\n")
	return b.String(), nil
}

func contentTypesForExport(content string) []string {
	switch strings.ToLower(strings.TrimSpace(content)) {
	case "posts", "post":
		return []string{"post"}
	case "pages", "page":
		return []string{"page"}
	case "attachment", "media":
		return []string{"attachment"}
	case "all":
		return []string{"post", "page", "attachment"}
	default:
		return []string{"post", "page", "attachment"}
	}
}

func wxrCDATA(s string) string {
	s = strings.ReplaceAll(s, "]]>", "]]]]><![CDATA[>")
	return "<![CDATA[" + s + "]]>"
}

func sanitizeExportKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-_")
	if out == "" {
		return "gopress"
	}
	return out
}

func mysqlDateToRFC822(mysql string) string {
	mysql = strings.TrimSpace(mysql)
	if mysql == "" || mysql == "0000-00-00 00:00:00" {
		return time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 +0000")
	}
	t, err := time.Parse("2006-01-02 15:04:05", mysql)
	if err != nil {
		return time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 +0000")
	}
	return t.UTC().Format("Mon, 02 Jan 2006 15:04:05 +0000")
}
