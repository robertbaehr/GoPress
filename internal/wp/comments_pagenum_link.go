package wp

import (
	"strconv"
	"strings"
)

// GetCommentsPagenumLink builds get_comments_pagenum_link (P3.152).
// permalink is the singular post URL; pagenum/maxPage mirror WP $pagenum/$max_page.
// pretty enables /comment-page-N/ paths (permalink_structure present).
// Applies get_comments_pagenum_link filter; always appends #comments.
func GetCommentsPagenumLink(c *Context, permalink string, pagenum, maxPage int, pretty bool) string {
	permalink = strings.TrimSpace(permalink)
	if permalink == "" {
		return ""
	}
	if pagenum < 1 {
		pagenum = 1
	}
	if maxPage < 0 {
		maxPage = 0
	}

	result := permalink
	defaultNewest := true
	if c != nil {
		if strings.EqualFold(strings.TrimSpace(optionStringValue(c, "default_comments_page", "newest")), "oldest") {
			defaultNewest = false
		}
	}

	needPage := false
	if defaultNewest {
		// WP: when newest, omit page segment only when pagenum == max_page.
		if maxPage == 0 || pagenum != maxPage {
			needPage = true
		}
	} else if pagenum > 1 {
		needPage = true
	}

	if needPage {
		if pretty {
			base := strings.TrimRight(result, "/")
			result = base + "/comment-page-" + strconv.Itoa(pagenum) + "/"
		} else {
			result = addQueryArg(result, "cpage", strconv.Itoa(pagenum))
		}
	}

	result += "#comments"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_comments_pagenum_link", result); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return result
}
