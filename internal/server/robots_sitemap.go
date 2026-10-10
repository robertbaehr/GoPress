package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveRobotsTXT writes a WordPress-compatible robots.txt (do_robots subset).
func serveRobotsTXT(w http.ResponseWriter, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	public := optionTruthy(c, "blog_public")
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	b.WriteString("Disallow: /wp-admin/\n")
	b.WriteString("Allow: /wp-admin/admin-ajax.php\n")
	if public && home != "" {
		b.WriteString("\nSitemap: ")
		b.WriteString(home)
		b.WriteString("/wp-sitemap.xml\n")
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// serveSitemapIndex writes /wp-sitemap.xml (sitemap index stub).
func serveSitemapIndex(w http.ResponseWriter, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	entries := []string{
		home + "/wp-sitemap-posts-post-1.xml",
		home + "/wp-sitemap-posts-page-1.xml",
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, loc := range entries {
		b.WriteString("\t<sitemap>\n\t\t<loc>")
		b.WriteString(xmlEscape(loc))
		b.WriteString("</loc>\n\t</sitemap>\n")
	}
	b.WriteString("</sitemapindex>\n")
	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// serveSitemapPosts writes a basic posts/pages sitemap page.
func serveSitemapPosts(w http.ResponseWriter, c *wp.Context, postType string) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	posts, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     postType,
		PostStatus:   "publish",
		PostsPerPage: 100,
	})
	if err != nil {
		http.Error(w, "sitemap: "+err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	if postType == "page" {
		// Front page when show_on_front=posts is the blog home URL.
		b.WriteString("\t<url>\n\t\t<loc>")
		b.WriteString(xmlEscape(home + "/"))
		b.WriteString("</loc>\n\t</url>\n")
	}
	for _, p := range posts {
		loc := postPermalink(home, p)
		if loc == "" {
			continue
		}
		b.WriteString("\t<url>\n\t\t<loc>")
		b.WriteString(xmlEscape(loc))
		b.WriteString("</loc>\n")
		if p.ModifiedGMT != "" && p.ModifiedGMT != "0000-00-00 00:00:00" {
			b.WriteString("\t\t<lastmod>")
			b.WriteString(xmlEscape(mysqlToW3C(p.ModifiedGMT)))
			b.WriteString("</lastmod>\n")
		}
		b.WriteString("\t</url>\n")
	}
	b.WriteString("</urlset>\n")
	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func postPermalink(home string, p wp.Post) string {
	if p.Type == "page" {
		if p.Name == "" {
			return ""
		}
		return home + "/" + p.Name + "/"
	}
	// Prefer pretty date permalinks when date is present.
	if len(p.Date) >= 10 && p.Name != "" {
		y, m, d := p.Date[0:4], p.Date[5:7], p.Date[8:10]
		return fmt.Sprintf("%s/%s/%s/%s/%s/", home, y, m, d, p.Name)
	}
	if p.ID > 0 {
		return home + "/?p=" + strconv.FormatInt(p.ID, 10)
	}
	return ""
}

func mysqlToW3C(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 19 {
		return s[0:10] + "T" + s[11:19] + "+00:00"
	}
	return s
}

func optionTruthy(c *wp.Context, name string) bool {
	if c == nil || c.Options == nil {
		return true
	}
	v := c.Options.GetOption(name, "1")
	switch t := v.(type) {
	case nil:
		return true
	case bool:
		return t
	case int64:
		return t != 0
	case int:
		return t != 0
	case string:
		s := strings.TrimSpace(t)
		return s != "" && s != "0" && !strings.EqualFold(s, "false")
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		return s != "" && s != "0"
	}
}
