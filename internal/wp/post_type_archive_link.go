package wp

import (
	"strings"
)

// GetPostTypeArchiveLink builds get_post_type_archive_link for "post" (P3.153).
// Other post types return "" until a post-type registry lands.
// Applies post_type_archive_link filter.
func GetPostTypeArchiveLink(c *Context, postType string) string {
	postType = strings.TrimSpace(postType)
	if postType == "" {
		return ""
	}
	out := ""
	switch postType {
	case "post":
		home := ""
		if c != nil {
			home = strings.TrimRight(optionStringBlogInfo(c, "home", optionStringBlogInfo(c, "siteurl", "")), "/")
		}
		showOnFront := ""
		pageForPosts := int64(0)
		if c != nil && c.Options != nil {
			showOnFront = strings.TrimSpace(optionValueString(c.Options.GetOption("show_on_front", "")))
			pageForPosts = optionInt64(c, "page_for_posts", 0)
		}
		if showOnFront == "page" && pageForPosts > 0 {
			if u := GetCanonicalURL(c, pageForPosts, 0, 0); strings.TrimSpace(u) != "" {
				out = u
			}
		}
		if out == "" {
			out = home + "/"
			if home == "" {
				out = "/"
			}
		}
	default:
		return ""
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("post_type_archive_link", out, postType); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}

// GetPostTypeArchiveFeedLink builds get_post_type_archive_feed_link (P3.153).
// Pretty path appends feed/ (and type when ≠ rss2). Applies post_type_archive_feed_link.
func GetPostTypeArchiveFeedLink(c *Context, postType, feed string) string {
	link := GetPostTypeArchiveLink(c, postType)
	if link == "" {
		return ""
	}
	feed = strings.ToLower(strings.TrimSpace(feed))
	if feed == "" {
		feed = "rss2"
	}
	base := strings.TrimRight(link, "/")
	out := ""
	if feed == "rss2" || feed == "rss" {
		out = base + "/feed/"
	} else {
		out = base + "/feed/" + feed + "/"
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("post_type_archive_feed_link", out, feed); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return out
}
