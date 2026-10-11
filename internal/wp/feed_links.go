package wp

import (
	"strings"
)

// FeedContentType returns the MIME type for a feed (feed_content_type, P3.147).
// Empty type uses the default feed (rss2 → application/rss+xml).
func FeedContentType(c *Context, feedType string) string {
	feedType = strings.TrimSpace(feedType)
	if feedType == "" {
		feedType = "rss2"
	}
	types := map[string]string{
		"rss":      "application/rss+xml",
		"rss2":     "application/rss+xml",
		"rss-http": "text/xml",
		"atom":     "application/atom+xml",
		"rdf":      "application/rdf+xml",
	}
	contentType, ok := types[feedType]
	if !ok {
		contentType = "application/octet-stream"
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("feed_content_type", contentType, feedType); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return contentType
}

// GetFeedLink builds a site feed permalink (get_feed_link, P3.147).
// Pretty: /feed/ or /comments/feed/; empty feed → default rss2 path without type segment.
// feed may be "", "rss2", "atom", "comments_rss2", "comments_atom", etc.
func GetFeedLink(c *Context, home, feed string) string {
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	if home == "" && c != nil {
		home = strings.TrimRight(optionStringBlogInfo(c, "home", optionStringBlogInfo(c, "siteurl", "")), "/")
	}
	if home == "" {
		return ""
	}
	feed = strings.TrimSpace(feed)
	out := ""
	if strings.HasPrefix(feed, "comments_") {
		rest := strings.TrimPrefix(feed, "comments_")
		if rest == "" || rest == "rss2" || rest == "rss" {
			out = home + "/comments/feed/"
		} else {
			out = home + "/comments/feed/" + rest + "/"
		}
	} else if feed == "" || feed == "rss2" || feed == "rss" {
		out = home + "/feed/"
	} else {
		out = home + "/feed/" + feed + "/"
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("feed_link", out, feed); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

// FeedAlternateLinkHTML builds one <link rel="alternate" …> feed tag with EscURL href (P3.147).
func FeedAlternateLinkHTML(c *Context, home, feed, titleAttr string) string {
	href := GetFeedLink(c, home, feed)
	if href == "" {
		return ""
	}
	ctype := FeedContentType(c, strings.TrimPrefix(feed, "comments_"))
	if ctype == "" {
		ctype = "application/rss+xml"
	}
	return `<link rel="alternate" type="` + ctype + `" title="` + titleAttr + `" href="` + EscURL(href) + `" />` + "\n"
}

// GetPostCommentsFeedLink builds a post comments feed permalink (get_post_comments_feed_link, P3.148).
// Pretty-permalink subset: {permalink}/feed/ or {permalink}/feed/{type}/ when type ≠ default rss2.
// Applies the post_comments_feed_link filter.
func GetPostCommentsFeedLink(c *Context, permalink, feed string) string {
	base := strings.TrimRight(strings.TrimSpace(permalink), "/")
	if base == "" {
		return ""
	}
	feed = strings.ToLower(strings.TrimSpace(feed))
	if feed == "" {
		feed = "rss2"
	}
	out := ""
	if feed == "rss2" || feed == "rss" {
		out = base + "/feed/"
	} else {
		out = base + "/feed/" + feed + "/"
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("post_comments_feed_link", out); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

// PostCommentsFeedLinkHTML builds an <a href="…">Comments Feed</a> tag with EscURL (P3.148).
// Mirrors post_comments_feed_link() display helper (anchor only; post_comments_feed_link_html filter).
func PostCommentsFeedLinkHTML(c *Context, permalink, feed, linkText string) string {
	url := GetPostCommentsFeedLink(c, permalink, feed)
	if url == "" {
		return ""
	}
	if strings.TrimSpace(linkText) == "" {
		linkText = "Comments Feed"
	}
	link := `<a href="` + EscURL(url) + `">` + linkText + `</a>`
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("post_comments_feed_link_html", link, int64(0), feed); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return link
}

// GetAuthorPostsURL builds an author archive permalink (get_author_posts_url subset, P3.149).
// Pretty: {home}/author/{nicename}/. Empty when author cannot be resolved.
func GetAuthorPostsURL(c *Context, home string, authorID int64) string {
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	if home == "" && c != nil {
		home = strings.TrimRight(optionStringBlogInfo(c, "home", optionStringBlogInfo(c, "siteurl", "")), "/")
	}
	if home == "" || authorID <= 0 || c == nil {
		return ""
	}
	u, err := c.GetUserBy("id", authorID)
	if err != nil || u == nil {
		return ""
	}
	slug := strings.TrimSpace(u.Nicename)
	if slug == "" {
		slug = strings.TrimSpace(u.Login)
	}
	if slug == "" {
		return ""
	}
	return home + "/author/" + slug + "/"
}

// GetAuthorFeedLink builds an author feed permalink (get_author_feed_link, P3.149).
// Pretty: {author_posts_url}feed/ or …/feed/{type}/ when type ≠ default rss2.
// Applies the author_feed_link filter.
func GetAuthorFeedLink(c *Context, home string, authorID int64, feed string) string {
	base := strings.TrimRight(GetAuthorPostsURL(c, home, authorID), "/")
	if base == "" {
		return ""
	}
	feed = strings.ToLower(strings.TrimSpace(feed))
	if feed == "" {
		feed = "rss2"
	}
	out := ""
	if feed == "rss2" || feed == "rss" {
		out = base + "/feed/"
	} else {
		out = base + "/feed/" + feed + "/"
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("author_feed_link", out, feed); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

// GetCategoryFeedLink builds a category feed permalink (get_category_feed_link, P3.150).
func GetCategoryFeedLink(c *Context, home string, catID int64, feed string) string {
	return GetTermFeedLink(c, home, catID, "category", feed)
}

// GetTagFeedLink builds a tag feed permalink (get_tag_feed_link, P3.150).
func GetTagFeedLink(c *Context, home string, tagID int64, feed string) string {
	return GetTermFeedLink(c, home, tagID, "post_tag", feed)
}

// GetTermFeedLink builds a term archive feed permalink (get_term_feed_link, P3.150).
// Pretty subset: /category/{slug}/feed/, /tag/{slug}/feed/, or /{taxonomy}/{slug}/feed/.
// Applies category_feed_link / tag_feed_link / taxonomy_feed_link filters.
func GetTermFeedLink(c *Context, home string, termID int64, taxonomy, feed string) string {
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	if home == "" && c != nil {
		home = strings.TrimRight(optionStringBlogInfo(c, "home", optionStringBlogInfo(c, "siteurl", "")), "/")
	}
	taxonomy = strings.TrimSpace(taxonomy)
	if home == "" || termID <= 0 || taxonomy == "" || c == nil || c.DB == nil {
		return ""
	}
	slug := termSlugForTaxonomy(c, termID, taxonomy)
	if slug == "" {
		return ""
	}
	basePath := ""
	switch taxonomy {
	case "category":
		basePath = "/category/" + slug
	case "post_tag":
		basePath = "/tag/" + slug
	default:
		basePath = "/" + taxonomy + "/" + slug
	}
	feed = strings.ToLower(strings.TrimSpace(feed))
	if feed == "" {
		feed = "rss2"
	}
	out := ""
	if feed == "rss2" || feed == "rss" {
		out = home + basePath + "/feed/"
	} else {
		out = home + basePath + "/feed/" + feed + "/"
	}
	if c.Hooks != nil {
		switch taxonomy {
		case "category":
			if v := c.Hooks.ApplyFilters("category_feed_link", out, feed); v != nil {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		case "post_tag":
			if v := c.Hooks.ApplyFilters("tag_feed_link", out, feed); v != nil {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		default:
			if v := c.Hooks.ApplyFilters("taxonomy_feed_link", out, feed, taxonomy); v != nil {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	return out
}

// GetSearchLink builds a search results permalink (get_search_link subset, P3.151).
// Pretty: {home}/search/{query}/ with spaces as +. Empty query → {home}/search/.
func GetSearchLink(c *Context, home, query string) string {
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	if home == "" && c != nil {
		home = strings.TrimRight(optionStringBlogInfo(c, "home", optionStringBlogInfo(c, "siteurl", "")), "/")
	}
	if home == "" {
		return ""
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return home + "/search/"
	}
	// WP uses rawurlencode then replaces %20 with +.
	enc := strings.ReplaceAll(pathEscapeSearch(q), "%20", "+")
	return home + "/search/" + enc + "/"
}

// GetSearchFeedLink builds a search results feed permalink (get_search_feed_link, P3.151).
// Pretty: {search_link}feed/ or …/feed/{type}/ when type ≠ default rss2.
// Applies search_feed_link filter with type "posts".
func GetSearchFeedLink(c *Context, home, query, feed string) string {
	base := strings.TrimRight(GetSearchLink(c, home, query), "/")
	if base == "" {
		return ""
	}
	feed = strings.ToLower(strings.TrimSpace(feed))
	if feed == "" {
		feed = "rss2"
	}
	out := ""
	if feed == "rss2" || feed == "rss" {
		out = base + "/feed/"
	} else {
		out = base + "/feed/" + feed + "/"
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("search_feed_link", out, feed, "posts"); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

// GetSearchCommentsFeedLink builds a search comments feed permalink (P3.151).
// Pretty subset: appends ?withcomments=1 to the search feed link.
// Applies search_feed_link filter with type "comments".
func GetSearchCommentsFeedLink(c *Context, home, query, feed string) string {
	out := GetSearchFeedLink(c, home, query, feed)
	if out == "" {
		return ""
	}
	if strings.Contains(out, "?") {
		out += "&withcomments=1"
	} else {
		out += "?withcomments=1"
	}
	if c != nil && c.Hooks != nil {
		feed = strings.ToLower(strings.TrimSpace(feed))
		if feed == "" {
			feed = "rss2"
		}
		if v := c.Hooks.ApplyFilters("search_feed_link", out, feed, "comments"); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

// pathEscapeSearch percent-encodes a search query for pretty /search/ paths.
func pathEscapeSearch(q string) string {
	var b strings.Builder
	for i := 0; i < len(q); i++ {
		c := q[i]
		// Unreserved + common path-safe chars WP keeps via rawurlencode of UTF-8.
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		const hex = "0123456789ABCDEF"
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0xf])
	}
	return b.String()
}

// termSlugForTaxonomy looks up a term slug when the term exists in taxonomy.
func termSlugForTaxonomy(c *Context, termID int64, taxonomy string) string {
	if c == nil || c.DB == nil || termID <= 0 || taxonomy == "" {
		return ""
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.slug FROM "+c.DB.Terms()+" t "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
			"WHERE t.term_id = %d AND tt.taxonomy = %s LIMIT 1",
		termID, taxonomy,
	)
	if err != nil {
		return ""
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return ""
	}
	return strings.TrimSpace(rowString(row["slug"]))
}
