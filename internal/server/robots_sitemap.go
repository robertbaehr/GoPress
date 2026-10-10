package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveRobotsTXT writes a WordPress-compatible robots.txt (do_robots subset).
// When blog_public=0, emit Disallow: / (P3.44 search-engine visibility off).
func serveRobotsTXT(w http.ResponseWriter, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	public := optionTruthy(c, "blog_public")
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	if !public {
		b.WriteString("Disallow: /\n")
	} else {
		b.WriteString("Disallow: /wp-admin/\n")
		b.WriteString("Allow: /wp-admin/admin-ajax.php\n")
		if home != "" {
			b.WriteString("\nSitemap: ")
			b.WriteString(home)
			b.WriteString("/wp-sitemap.xml\n")
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if home != "" {
		w.Header().Set("Link", fmt.Sprintf(`<%s/wp-json/>; rel="https://api.w.org/"`, home))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// serveSitemapIndex writes /wp-sitemap.xml (WP sitemap index; P3.14).
func serveSitemapIndex(w http.ResponseWriter, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	entries := []string{
		home + "/wp-sitemap-posts-post-1.xml",
		home + "/wp-sitemap-posts-page-1.xml",
		home + "/wp-sitemap-taxonomies-category-1.xml",
		home + "/wp-sitemap-taxonomies-post_tag-1.xml",
		home + "/wp-sitemap-taxonomies-post_format-1.xml",
		home + "/wp-sitemap-users-1.xml",
	}
	// Compact XML matches WP WP_Sitemaps_Renderer output (no pretty-print).
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, loc := range entries {
		b.WriteString("<sitemap><loc>")
		b.WriteString(xmlEscape(loc))
		b.WriteString("</loc></sitemap>")
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

// serveSitemapTaxonomy writes a taxonomy terms sitemap page (P3.14).
func serveSitemapTaxonomy(w http.ResponseWriter, c *wp.Context, taxonomy string) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	if c != nil && c.DB != nil {
		q, args, err := c.DB.Prepare(
			"SELECT t.slug, tt.parent FROM "+c.DB.Terms()+" t "+
				"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
				"WHERE tt.taxonomy = %s AND tt.count > 0 ORDER BY t.name ASC",
			taxonomy,
		)
		if err == nil {
			if rows, err := c.DB.GetResults(q, args...); err == nil {
				base := taxonomyBase(taxonomy)
				for _, row := range rows {
					slug := strings.TrimSpace(fmt.Sprint(row["slug"]))
					if slug == "" {
						continue
					}
					loc := home + "/" + base + "/" + slug + "/"
					// Hierarchical category paths are approximated via slug only when parent chain
					// is not resolved; WP emits full paths — improve when term cache is richer.
					if taxonomy == "category" {
						loc = categorySitemapLoc(c, home, row)
					}
					b.WriteString("<url><loc>")
					b.WriteString(xmlEscape(loc))
					b.WriteString("</loc></url>")
				}
			}
		}
	}
	b.WriteString("</urlset>\n")
	w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

func taxonomyBase(taxonomy string) string {
	switch taxonomy {
	case "category":
		return "category"
	case "post_tag":
		return "tag"
	case "post_format":
		return "type"
	default:
		return taxonomy
	}
}

func categorySitemapLoc(c *wp.Context, home string, row map[string]any) string {
	slug := strings.TrimSpace(fmt.Sprint(row["slug"]))
	if slug == "" {
		return home + "/category/"
	}
	// Build parent/child path when possible.
	parent := int64(0)
	switch t := row["parent"].(type) {
	case int64:
		parent = t
	case int:
		parent = int64(t)
	case string:
		parent, _ = strconv.ParseInt(t, 10, 64)
	}
	parts := []string{slug}
	seen := map[int64]bool{}
	for parent > 0 && !seen[parent] && c != nil && c.DB != nil {
		seen[parent] = true
		q, args, err := c.DB.Prepare(
			"SELECT t.slug, tt.parent FROM "+c.DB.Terms()+" t "+
				"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
				"WHERE tt.term_taxonomy_id = %d OR t.term_id = %d LIMIT 1",
			parent, parent,
		)
		if err != nil {
			break
		}
		prow, err := c.DB.GetRow(q, args...)
		if err != nil || prow == nil {
			break
		}
		pslug := strings.TrimSpace(fmt.Sprint(prow["slug"]))
		if pslug != "" {
			parts = append([]string{pslug}, parts...)
		}
		switch t := prow["parent"].(type) {
		case int64:
			parent = t
		case int:
			parent = int64(t)
		case string:
			parent, _ = strconv.ParseInt(t, 10, 64)
		default:
			parent = 0
		}
	}
	return home + "/category/" + strings.Join(parts, "/") + "/"
}

// serveSitemapUsers writes an author sitemap page (P3.14).
func serveSitemapUsers(w http.ResponseWriter, c *wp.Context) {
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	if c != nil && c.DB != nil {
		// Authors who have published posts (WP users provider subset).
		q, args, err := c.DB.Prepare(
			"SELECT DISTINCT u.user_nicename FROM " + c.DB.Users() + " u " +
				"INNER JOIN " + c.DB.Posts() + " p ON p.post_author = u.ID " +
				"WHERE p.post_type = 'post' AND p.post_status = 'publish' " +
				"AND u.user_nicename <> '' ORDER BY u.user_nicename ASC",
		)
		if err == nil {
			if rows, err := c.DB.GetResults(q, args...); err == nil {
				for _, row := range rows {
					nicename := strings.TrimSpace(fmt.Sprint(row["user_nicename"]))
					if nicename == "" {
						continue
					}
					b.WriteString("<url><loc>")
					b.WriteString(xmlEscape(home + "/author/" + nicename + "/"))
					b.WriteString("</loc></url>")
				}
			}
		}
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
