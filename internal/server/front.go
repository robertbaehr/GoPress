package server

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/blocks"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// Front is the minimal front controller / shim renderer (P3.3+).
// Theme PHP is out of scope; this emits small HTML documents for home and singular.
type Front struct{}

// ServeFront implements FrontHandler.
func (f *Front) ServeFront(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil {
		http.Error(w, "GoPress context unavailable\n", http.StatusServiceUnavailable)
		return
	}
	c.AuthenticateRequest()

	rules, err := c.LoadRewriteRules()
	if err != nil {
		http.Error(w, "rewrite rules: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	parsed := c.ParseRequest(rules)

	if isFeedVars(parsed.QueryVars) {
		f.renderFeedRSS2(w, c)
		return
	}

	if parsed.IsHome {
		f.renderHome(w, c)
		return
	}

	if isSingularVars(parsed.QueryVars) {
		f.renderSingular(w, c, parsed.QueryVars)
		return
	}

	// Category/tag archives: list posts in term (shim until full archive templates).
	if cat := parsed.QueryVars["category_name"]; cat != "" {
		f.renderTermArchive(w, c, "category", cat)
		return
	}
	if tag := parsed.QueryVars["tag"]; tag != "" {
		f.renderTermArchive(w, c, "post_tag", tag)
		return
	}

	// redirect_canonical lite: unmatched pretty paths whose last segment is a
	// published post slug (e.g. /blog/uncategorized/hello-world/ → date permalink).
	if parsed.Request != "" {
		if f.tryCanonicalPostRedirect(w, r, c, parsed.Request) {
			return
		}
	}

	// Later tasks: date/author archives.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "GoPress: front route not implemented (request=%q rule=%q)\n",
		parsed.Request, parsed.MatchedRule)
}

func isFeedVars(vars map[string]string) bool {
	if vars == nil {
		return false
	}
	feed := vars["feed"]
	if feed == "" {
		return false
	}
	// WP default feed is rss2; "feed" is the permalink alias for the default.
	switch strings.ToLower(feed) {
	case "feed", "rss", "rss2":
		return true
	default:
		return false
	}
}

func isSingularVars(vars map[string]string) bool {
	if vars == nil {
		return false
	}
	for _, k := range []string{"p", "page_id", "name", "pagename", "attachment_id", "attachment"} {
		if v := vars[k]; v != "" {
			return true
		}
	}
	return false
}

func (f *Front) renderHome(w http.ResponseWriter, c *wp.Context) {
	// P5.5: prefer block theme home/index template (DB or theme HTML) over shim.
	if tpl, ok := wp.ResolveBlockTemplate(c, "home", "index"); ok && strings.TrimSpace(tpl) != "" {
		f.renderBlockTemplate(w, c, tpl)
		return
	}
	f.renderHomeShim(w, c)
}

// renderBlockTemplate parses and renders a block theme template document.
func (f *Front) renderBlockTemplate(w http.ResponseWriter, c *wp.Context, template string) {
	blogname := optionString(c, "blogname", "GoPress")
	body := blocks.RenderWith(blockRenderEnv(c), template)
	safe := wp.KsesPost(body)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "wp-template")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString(safe)
	b.WriteString("\n</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderHomeShim(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := latestPosts(c, 5)
	if err != nil {
		http.Error(w, "posts: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body")
	if cls := wp.BodyClassWithAdminBar(c, ""); cls != "" {
		b.WriteString(` class="`)
		b.WriteString(html.EscapeString(cls))
		b.WriteString(`"`)
	}
	b.WriteString(">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</h1>\n")
	if c.CurrentUser != nil {
		b.WriteString("<p class=\"user\">Logged in as ")
		b.WriteString(html.EscapeString(c.CurrentUser.Login))
		b.WriteString("</p>\n")
	}
	b.WriteString("<ul class=\"posts\">\n")
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) renderSingular(w http.ResponseWriter, c *wp.Context, vars map[string]string) {
	post, err := loadSingularPost(c, vars)
	if err != nil {
		http.Error(w, "post: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	if post == nil {
		f.render404(w, c)
		return
	}

	blogname := optionString(c, "blogname", "GoPress")
	content := post.Content
	if post.Type == "attachment" {
		if content == "" {
			if url := wp.GetAttachmentURL(c, post.ID); url != "" {
				content = `<p class="attachment"><a href="` + html.EscapeString(url) + `">` +
					html.EscapeString(post.Title) + `</a></p>`
			}
		}
	} else if blocks.HasBlocks(content) {
		content = blocks.Render(content)
	}
	// the_content: do_shortcode (P2.11) before display.
	content = wp.DoShortcode(c, content)
	safeContent := wp.KsesPost(content)

	bodyClass := "singular"
	if post.Type == "attachment" {
		bodyClass = "singular attachment"
	}

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString(" &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, bodyClass)))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<article id=\"post-")
	b.WriteString(strconv.FormatInt(post.ID, 10))
	b.WriteString("\" class=\"post type-")
	b.WriteString(html.EscapeString(post.Type))
	b.WriteString("\">\n")
	b.WriteString("<h1 class=\"entry-title\">")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString("</h1>\n")
	b.WriteString("<div class=\"entry-content\">\n")
	b.WriteString(safeContent)
	b.WriteString("\n</div>\n</article>\n")
	if post.Type == "post" || post.Type == "page" {
		f.writeCommentsSection(&b, c, post)
	}
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) writeCommentsSection(b *strings.Builder, c *wp.Context, post *singularPost) {
	comments, err := wp.ListApprovedComments(c, post.ID)
	if err != nil {
		comments = nil
	}
	b.WriteString("<section id=\"comments\" class=\"comments-area\">\n")
	b.WriteString("<h2 class=\"comments-title\">Comments</h2>\n")
	if len(comments) == 0 {
		b.WriteString("<p class=\"no-comments\">No comments yet.</p>\n")
	} else {
		b.WriteString("<ol class=\"comment-list\">\n")
		for _, cm := range comments {
			b.WriteString("<li id=\"comment-")
			b.WriteString(strconv.FormatInt(cm.ID, 10))
			b.WriteString("\" class=\"comment\">\n")
			b.WriteString("<article class=\"comment-body\">\n")
			b.WriteString("<footer class=\"comment-meta\"><b class=\"fn\">")
			b.WriteString(html.EscapeString(cm.Author))
			b.WriteString("</b></footer>\n")
			b.WriteString("<div class=\"comment-content\">\n")
			b.WriteString(wp.KsesData(cm.Content))
			b.WriteString("\n</div>\n</article>\n</li>\n")
		}
		b.WriteString("</ol>\n")
	}
	// Comment form posts to wp-comments-post.php.
	b.WriteString("<div id=\"respond\" class=\"comment-respond\">\n")
	b.WriteString("<h3 id=\"reply-title\" class=\"comment-reply-title\">Leave a Reply</h3>\n")
	b.WriteString("<form action=\"/wp-comments-post.php\" method=\"post\" id=\"commentform\" class=\"comment-form\">\n")
	b.WriteString("<input type=\"hidden\" name=\"comment_post_ID\" value=\"")
	b.WriteString(strconv.FormatInt(post.ID, 10))
	b.WriteString("\" id=\"comment_post_ID\">\n")
	b.WriteString("<input type=\"hidden\" name=\"comment_parent\" id=\"comment_parent\" value=\"0\">\n")
	if c == nil || c.CurrentUser == nil {
		b.WriteString("<p class=\"comment-form-author\"><label for=\"author\">Name</label> ")
		b.WriteString("<input id=\"author\" name=\"author\" type=\"text\" required></p>\n")
		b.WriteString("<p class=\"comment-form-email\"><label for=\"email\">Email</label> ")
		b.WriteString("<input id=\"email\" name=\"email\" type=\"email\" required></p>\n")
		b.WriteString("<p class=\"comment-form-url\"><label for=\"url\">Website</label> ")
		b.WriteString("<input id=\"url\" name=\"url\" type=\"url\"></p>\n")
	}
	b.WriteString("<p class=\"comment-form-comment\"><label for=\"comment\">Comment</label> ")
	b.WriteString("<textarea id=\"comment\" name=\"comment\" cols=\"45\" rows=\"8\" required></textarea></p>\n")
	b.WriteString("<p class=\"form-submit\"><input name=\"submit\" type=\"submit\" id=\"submit\" class=\"submit\" value=\"Post Comment\"></p>\n")
	b.WriteString("</form>\n</div>\n</section>\n")
}

func (f *Front) renderFeedRSS2(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	desc := optionString(c, "blogdescription", "")
	posts, err := latestFeedPosts(c, 10)
	if err != nil {
		http.Error(w, "feed: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}

	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString(`<rss version="2.0">` + "\n")
	b.WriteString("<channel>\n")
	b.WriteString("<title>")
	b.WriteString(xmlEscape(blogname))
	b.WriteString("</title>\n")
	b.WriteString("<link>")
	b.WriteString(xmlEscape(home + "/"))
	b.WriteString("</link>\n")
	b.WriteString("<description>")
	b.WriteString(xmlEscape(desc))
	b.WriteString("</description>\n")
	b.WriteString("<lastBuildDate>")
	b.WriteString(time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("</lastBuildDate>\n")
	b.WriteString("<language>en-US</language>\n")
	for _, p := range posts {
		link := fmt.Sprintf("%s/?p=%d", home, p.ID)
		b.WriteString("<item>\n")
		b.WriteString("<title>")
		b.WriteString(xmlEscape(p.Title))
		b.WriteString("</title>\n")
		b.WriteString("<link>")
		b.WriteString(xmlEscape(link))
		b.WriteString("</link>\n")
		b.WriteString("<guid isPermaLink=\"false\">")
		b.WriteString(xmlEscape(link))
		b.WriteString("</guid>\n")
		if p.Date != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", p.Date); err == nil {
				b.WriteString("<pubDate>")
				b.WriteString(t.Format(time.RFC1123Z))
				b.WriteString("</pubDate>\n")
			}
		}
		b.WriteString("<description><![CDATA[")
		b.WriteString(wp.KsesPost(p.Content))
		b.WriteString("]]></description>\n")
		b.WriteString("</item>\n")
	}
	b.WriteString("</channel>\n</rss>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "application/rss+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

type feedPost struct {
	ID      int64
	Title   string
	Content string
	Date    string
}

func latestFeedPosts(c *wp.Context, limit int) ([]feedPost, error) {
	if limit <= 0 {
		limit = 10
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostType: "post", PostStatus: "publish", PostsPerPage: limit})
	if err != nil {
		return nil, err
	}
	out := make([]feedPost, 0, len(posts))
	for _, p := range posts {
		out = append(out, feedPost{ID: p.ID, Title: p.Title, Content: p.Content, Date: p.Date})
	}
	return out, nil
}

// tryCanonicalPostRedirect issues a 301 to the post's pretty permalink when the
// last path segment matches a published post name (WP redirect_canonical subset).
func (f *Front) tryCanonicalPostRedirect(w http.ResponseWriter, r *http.Request, c *wp.Context, request string) bool {
	parts := strings.Split(strings.Trim(request, "/"), "/")
	if len(parts) == 0 {
		return false
	}
	slug := parts[len(parts)-1]
	if slug == "" {
		return false
	}
	post, err := wp.GetPost(c, wp.QueryArgs{Name: slug, PostType: "post", PostStatus: "publish"})
	if err != nil || post == nil {
		return false
	}
	// Prefer date permalink when post_date is available.
	target := ""
	if post.Date != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", post.Date); err == nil {
			target = fmt.Sprintf("/%04d/%02d/%02d/%s/", t.Year(), int(t.Month()), t.Day(), post.Name)
		}
	}
	if target == "" {
		target = "/?p=" + strconv.FormatInt(post.ID, 10)
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
	return true
}

func (f *Front) renderTermArchive(w http.ResponseWriter, c *wp.Context, taxonomy, slug string) {
	blogname := optionString(c, "blogname", "GoPress")
	posts, err := latestPosts(c, 10)
	if err != nil {
		http.Error(w, "archive: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(slug))
	b.WriteString(" &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "archive")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(taxonomy))
	b.WriteString(": ")
	b.WriteString(html.EscapeString(slug))
	b.WriteString("</h1>\n<ul class=\"posts\">\n")
	for _, p := range posts {
		b.WriteString("<li><a href=\"?p=")
		b.WriteString(strconv.FormatInt(p.ID, 10))
		b.WriteString("\">")
		b.WriteString(html.EscapeString(p.Title))
		b.WriteString("</a></li>\n")
	}
	b.WriteString("</ul>\n</body>\n</html>\n")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func (f *Front) render404(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Page not found &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"")
	b.WriteString(html.EscapeString(wp.BodyClassWithAdminBar(c, "error404")))
	b.WriteString("\">\n")
	b.WriteString(wp.AdminBarHTML(c))
	b.WriteString("<h1>Page not found</h1>\n")
	b.WriteString("<p>The requested content could not be found.</p>\n")
	b.WriteString("</body>\n</html>\n")

	setFrontAPILink(w, c)
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(b.String()))
}

// setFrontAPILink mirrors rest_output_link_header (rel=https://api.w.org/).
func setFrontAPILink(w http.ResponseWriter, c *wp.Context) {
	siteURL := strings.TrimRight(optionString(c, "siteurl", ""), "/")
	home := strings.TrimRight(optionString(c, "home", siteURL), "/")
	if home == "" {
		home = siteURL
	}
	if home == "" {
		return
	}
	w.Header().Set("Link", fmt.Sprintf(`<%s/wp-json/>; rel="https://api.w.org/"`, home))
}

type postListItem struct {
	ID    int64
	Title string
}

type singularPost struct {
	ID      int64
	Title   string
	Content string
	Type    string
	Name    string
	Status  string
}

func loadSingularPost(c *wp.Context, vars map[string]string) (*singularPost, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if idStr := vars["attachment_id"]; idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			return nil, nil
		}
		return singularFromQuery(c, wp.QueryArgs{P: id, PostType: "attachment", PostStatus: "inherit"})
	}
	if slug := vars["attachment"]; slug != "" {
		return singularFromQuery(c, wp.QueryArgs{Name: slug, PostType: "attachment", PostStatus: "inherit"})
	}
	if idStr := vars["p"]; idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			return nil, nil
		}
		return singularFromQuery(c, wp.QueryArgs{P: id, PostType: "post"})
	}
	if idStr := vars["page_id"]; idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			return nil, nil
		}
		return singularFromQuery(c, wp.QueryArgs{P: id, PostType: "page"})
	}
	if name := vars["name"]; name != "" {
		return singularFromQuery(c, wp.QueryArgs{Name: name, PostType: "post"})
	}
	if pagename := vars["pagename"]; pagename != "" {
		return singularFromQuery(c, wp.QueryArgs{PageName: pagename, PostType: "page"})
	}
	return nil, nil
}

func singularFromQuery(c *wp.Context, args wp.QueryArgs) (*singularPost, error) {
	p, err := wp.GetPost(c, args)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return &singularPost{
		ID:      p.ID,
		Title:   p.Title,
		Content: p.Content,
		Type:    p.Type,
		Name:    p.Name,
		Status:  p.Status,
	}, nil
}

func latestPosts(c *wp.Context, limit int) ([]postListItem, error) {
	if limit <= 0 {
		limit = 5
	}
	posts, err := wp.GetPosts(c, wp.QueryArgs{PostType: "post", PostStatus: "publish", PostsPerPage: limit})
	if err != nil {
		return nil, err
	}
	out := make([]postListItem, 0, len(posts))
	for _, p := range posts {
		out = append(out, postListItem{ID: p.ID, Title: p.Title})
	}
	return out, nil
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

// blockRenderEnv wires dynamic block resolvers (template parts, navigation).
func blockRenderEnv(c *wp.Context) *blocks.Env {
	return &blocks.Env{
		LoadTemplatePart: func(slug string) (string, bool) {
			return wp.ResolveTemplatePart(c, slug)
		},
		LoadNavigation: func(ref int64) (string, bool) {
			return wp.LoadNavigationContent(c, ref)
		},
		ListPages: func() []blocks.NavLink {
			pages, err := wp.ListPublishedPages(c, 50)
			if err != nil {
				return nil
			}
			out := make([]blocks.NavLink, 0, len(pages))
			for _, p := range pages {
				out = append(out, blocks.NavLink{
					Label: p.Title,
					URL:   wp.PagePermalink(c, p),
				})
			}
			return out
		},
	}
}
