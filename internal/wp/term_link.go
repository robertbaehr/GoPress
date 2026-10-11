package wp

import (
	"fmt"
	"strings"
)

// GetCategoryLink builds a category archive permalink (get_category_link, P3.162).
func GetCategoryLink(c *Context, catID int64) string {
	return GetTermLink(c, catID, "category")
}

// GetTagLink builds a post_tag archive permalink (get_tag_link, P3.162).
func GetTagLink(c *Context, tagID int64) string {
	return GetTermLink(c, tagID, "post_tag")
}

// GetTermLink builds a term archive permalink (get_term_link subset, P3.162).
// Pretty paths: /category/{slug}/, /tag/{slug}/, /{taxonomy}/{slug}/ when
// permalink_structure is set; otherwise ?cat=ID / ?tag=slug / ?taxonomy=&term=.
// Applies category_link / tag_link then term_link filters. Returns "" when missing.
func GetTermLink(c *Context, termID int64, taxonomy string) string {
	taxonomy = strings.TrimSpace(taxonomy)
	if c == nil || c.DB == nil || termID <= 0 || taxonomy == "" {
		return ""
	}
	slug := termSlugForTaxonomy(c, termID, taxonomy)
	if slug == "" {
		return ""
	}
	home := strings.TrimRight(HomeURL(c, "", ""), "/")
	if home == "" {
		return ""
	}

	var link string
	if usingDatePermalinks(c) {
		switch taxonomy {
		case "category":
			link = home + "/category/" + slug + "/"
		case "post_tag":
			link = home + "/tag/" + slug + "/"
		default:
			link = home + "/" + taxonomy + "/" + slug + "/"
		}
	} else {
		switch taxonomy {
		case "category":
			link = home + "/?cat=" + fmt.Sprintf("%d", termID)
		case "post_tag":
			link = home + "/?tag=" + slug
		default:
			link = home + "/?taxonomy=" + taxonomy + "&term=" + slug
		}
	}

	if c.Hooks != nil {
		switch taxonomy {
		case "category":
			if v := c.Hooks.ApplyFilters("category_link", link, termID); v != nil {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					link = strings.TrimSpace(s)
				}
			}
		case "post_tag":
			if v := c.Hooks.ApplyFilters("tag_link", link, termID); v != nil {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					link = strings.TrimSpace(s)
				}
			}
		}
		if v := c.Hooks.ApplyFilters("term_link", link, termID, taxonomy); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return link
}
