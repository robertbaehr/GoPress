package server

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Front is the minimal front controller / shim renderer (P3.3).
// Theme PHP is out of scope; this emits a small HTML document for home.
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

	if parsed.IsHome {
		f.renderHome(w, c)
		return
	}

	// Later tasks: singular / archive / REST via rewrite. Stub for now.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "GoPress: front route not implemented (request=%q rule=%q)\n",
		parsed.Request, parsed.MatchedRule)
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

type postListItem struct {
	ID    int64
	Title string
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
