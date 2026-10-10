package server

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	// Later tasks: archive via rewrite.
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
	for _, k := range []string{"p", "page_id", "name", "pagename"} {
		if v := vars[k]; v != "" {
			return true
		}
	}
	return false
}

func (f *Front) renderHome(w http.ResponseWriter, c *wp.Context) {
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
	b.WriteString("</title>\n</head>\n<body>\n")
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	safeContent := wp.KsesPost(post.Content)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(post.Title))
	b.WriteString(" &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"singular\">\n")
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
	b.WriteString("\n</div>\n</article>\n</body>\n</html>\n")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
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

func (f *Front) render404(w http.ResponseWriter, c *wp.Context) {
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Page not found &#8212; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString("</title>\n</head>\n<body class=\"error404\">\n")
	b.WriteString("<h1>Page not found</h1>\n")
	b.WriteString("<p>The requested content could not be found.</p>\n")
	b.WriteString("</body>\n</html>\n")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(b.String()))
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
