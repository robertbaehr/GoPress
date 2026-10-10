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
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 10
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_title, post_content, post_date FROM "+c.DB.Posts()+
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
	out := make([]feedPost, 0, len(rows))
	for _, row := range rows {
		out = append(out, feedPost{
			ID:      asInt64(row["ID"]),
			Title:   asString(row["post_title"]),
			Content: asString(row["post_content"]),
			Date:    asString(row["post_date"]),
		})
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
		return getPublishedByID(c, id, "post")
	}
	if idStr := vars["page_id"]; idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			return nil, nil
		}
		return getPublishedByID(c, id, "page")
	}
	if name := vars["name"]; name != "" {
		// Pretty permalinks may include a trailing page number segment; use final slug.
		slug := lastPathSegment(name)
		return getPublishedBySlug(c, slug, "post")
	}
	if pagename := vars["pagename"]; pagename != "" {
		slug := lastPathSegment(pagename)
		return getPublishedBySlug(c, slug, "page")
	}
	return nil, nil
}

func getPublishedByID(c *wp.Context, id int64, postType string) (*singularPost, error) {
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_title, post_content, post_type, post_name, post_status FROM "+c.DB.Posts()+
			" WHERE ID = %d AND post_type = %s AND post_status = %s LIMIT 1",
		id, postType, "publish",
	)
	if err != nil {
		return nil, err
	}
	return scanSingularRow(c, q, args...)
}

func getPublishedBySlug(c *wp.Context, slug, postType string) (*singularPost, error) {
	if slug == "" {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_title, post_content, post_type, post_name, post_status FROM "+c.DB.Posts()+
			" WHERE post_name = %s AND post_type = %s AND post_status = %s LIMIT 1",
		slug, postType, "publish",
	)
	if err != nil {
		return nil, err
	}
	return scanSingularRow(c, q, args...)
}

func scanSingularRow(c *wp.Context, q string, args ...any) (*singularPost, error) {
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return &singularPost{
		ID:      asInt64(row["ID"]),
		Title:   asString(row["post_title"]),
		Content: asString(row["post_content"]),
		Type:    asString(row["post_type"]),
		Name:    asString(row["post_name"]),
		Status:  asString(row["post_status"]),
	}, nil
}

func lastPathSegment(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func latestPosts(c *wp.Context, limit int) ([]postListItem, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 5
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_title FROM "+c.DB.Posts()+
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
	out := make([]postListItem, 0, len(rows))
	for _, row := range rows {
		id := asInt64(row["ID"])
		title, _ := row["post_title"].(string)
		out = append(out, postListItem{ID: id, Title: title})
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
