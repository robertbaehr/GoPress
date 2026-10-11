// get_bloginfo / bloginfo subset (P3.53 / P3.113).
package wp

import (
	"fmt"
	"strings"
)

// BlogInfo returns site info for show (get_bloginfo).
// filter is "raw" (default) or "display" (applies bloginfo / bloginfo_url).
func BlogInfo(c *Context, show, filter string) string {
	show = strings.TrimSpace(show)
	filter = strings.TrimSpace(filter)
	if filter == "" {
		filter = "raw"
	}

	var output string
	switch show {
	case "home", "siteurl", "url":
		output = strings.TrimRight(HomeURL(c, "", ""), "/") + "/"
		// HomeURL may already include trailing slash; keep WP-ish home_url().
		if output == "/" {
			output = HomeURL(c, "/", "")
		}
	case "wpurl":
		output = strings.TrimRight(SiteURL(c, "", ""), "/") + "/"
		if output == "/" {
			output = SiteURL(c, "/", "")
		}
	case "description":
		output = optionStringBlogInfo(c, "blogdescription", "")
	case "rdf_url":
		output = strings.TrimRight(HomeURL(c, "", ""), "/") + "/feed/rdf/"
	case "rss_url":
		output = strings.TrimRight(HomeURL(c, "", ""), "/") + "/feed/rss/"
	case "rss2_url":
		output = strings.TrimRight(HomeURL(c, "", ""), "/") + "/feed/"
	case "atom_url":
		output = strings.TrimRight(HomeURL(c, "", ""), "/") + "/feed/atom/"
	case "comments_atom_url":
		output = strings.TrimRight(HomeURL(c, "", ""), "/") + "/comments/feed/atom/"
	case "comments_rss2_url":
		output = strings.TrimRight(HomeURL(c, "", ""), "/") + "/comments/feed/"
	case "pingback_url":
		output = strings.TrimRight(SiteURL(c, "", ""), "/") + "/xmlrpc.php"
	case "admin_email":
		output = optionStringBlogInfo(c, "admin_email", "")
	case "charset":
		output = optionStringBlogInfo(c, "blog_charset", "")
		if output == "" {
			output = "UTF-8"
		}
	case "html_type":
		output = optionStringBlogInfo(c, "html_type", "")
		if output == "" {
			output = "text/html"
		}
	case "version":
		output = Version
	case "language":
		output = BlogInfoLanguage(c)
	case "name", "":
		output = optionStringBlogInfo(c, "blogname", "")
	default:
		output = optionStringBlogInfo(c, "blogname", "")
	}

	if filter == "display" && c != nil && c.Hooks != nil {
		isURL := strings.Contains(show, "url") || strings.Contains(show, "directory") || strings.Contains(show, "home")
		if isURL {
			if v := c.Hooks.ApplyFilters("bloginfo_url", output, show); v != nil {
				output = fmt.Sprint(v)
			}
		} else {
			if v := c.Hooks.ApplyFilters("bloginfo", output, show); v != nil {
				output = fmt.Sprint(v)
			}
		}
	}
	return output
}

func optionStringBlogInfo(c *Context, key, def string) string {
	if c == nil || c.Options == nil {
		return def
	}
	v := c.Options.GetOption(key, def)
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return def
		}
		return t
	case nil:
		return def
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		if s == "" || s == "<nil>" {
			return def
		}
		return s
	}
}
