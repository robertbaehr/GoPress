package wp

import (
	"fmt"
	"strconv"
	"strings"
)

const wpTitleSepToken = "%WP_TITLE_SEP%"

// GetWPTitle builds the classic theme document title fragment (wp_title, P3.125).
// sep defaults to "»" when empty; seplocation "right" places the separator on
// the right. Applies wp_title_parts then wp_title filters. Does not echo.
func GetWPTitle(c *Context, sep, seplocation string) string {
	if sep == "" {
		sep = "»"
	}
	title := wpTitlePartsRaw(c)
	prefix := ""
	var parts []string
	if title != "" {
		prefix = " " + sep + " "
		parts = strings.Split(title, wpTitleSepToken)
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_title_parts", parts); v != nil {
			switch t := v.(type) {
			case []string:
				parts = t
			case []any:
				parts = make([]string, 0, len(t))
				for _, x := range t {
					parts = append(parts, fmt.Sprint(x))
				}
			}
		}
	}
	var out string
	if strings.EqualFold(seplocation, "right") {
		// Reverse breadcrumb; separator on the right.
		for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
			parts[i], parts[j] = parts[j], parts[i]
		}
		out = strings.Join(parts, " "+sep+" ") + prefix
	} else {
		out = prefix + strings.Join(parts, " "+sep+" ")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_title", out, sep, seplocation); v != nil {
			if s, ok := v.(string); ok {
				out = s
			}
		}
	}
	return out
}

// wpTitlePartsRaw assembles the pre-filter title using temporary separators.
func wpTitlePartsRaw(c *Context) string {
	if c == nil || c.Scope == nil {
		return ""
	}
	s := c.Scope

	// Singular / posts page (not front).
	if s.IsSingle || (s.IsHome && !s.IsFrontPage) || (s.IsPage && !s.IsFrontPage) {
		if title := queriedPostTitle(c); title != "" {
			return title
		}
	}
	if s.IsCategory && s.CategorySlug != "" {
		return s.CategorySlug
	}
	if s.IsTag && s.TagSlug != "" {
		return s.TagSlug
	}
	if s.IsAuthor {
		if s.AuthorName != "" {
			return s.AuthorName
		}
		if s.AuthorID > 0 {
			if u, err := c.GetUserBy("id", s.AuthorID); err == nil && u != nil {
				if u.DisplayName != "" {
					return u.DisplayName
				}
				return u.Login
			}
		}
	}
	if s.IsDate {
		if s.IsYear || s.DateYear > 0 {
			title := strconv.Itoa(s.DateYear)
			if s.DateMonth > 0 {
				title += wpTitleSepToken + monthName(s.DateMonth)
			}
			if s.DateDay > 0 {
				title += wpTitleSepToken + fmt.Sprintf("%02d", s.DateDay)
			}
			return title
		}
	}
	if s.IsSearch {
		q := strings.TrimSpace(s.SearchQuery)
		return "Search Results " + wpTitleSepToken + " " + WPStripAllTags(q, false)
	}
	if s.Is404 {
		return "Page not found"
	}
	return ""
}

func queriedPostTitle(c *Context) string {
	if c == nil || c.Scope == nil || c.Scope.QueriedObjectID <= 0 {
		return ""
	}
	post, err := GetPostByID(c, c.Scope.QueriedObjectID)
	if err != nil || post == nil {
		return ""
	}
	return post.Title
}

func monthName(m int) string {
	names := []string{
		"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	if m < 1 || m > 12 {
		return strconv.Itoa(m)
	}
	return names[m]
}
