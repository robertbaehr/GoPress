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
