package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// GetDocumentTitle builds the modern document title (wp_get_document_title, P3.126).
// Honors pre_get_document_title, document_title_parts, document_title_separator,
// and document_title filters. Returns plain text (caller escapes for HTML).
func GetDocumentTitle(c *Context) string {
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("pre_get_document_title", ""); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}

	parts := map[string]string{
		"title": "",
	}
	s := (*RequestScope)(nil)
	if c != nil {
		s = c.Scope
	}

	switch {
	case s != nil && s.Is404:
		parts["title"] = "Page not found"
	case s != nil && s.IsSearch:
		q := ""
		if s != nil {
			q = strings.TrimSpace(s.SearchQuery)
		}
		if q != "" {
			parts["title"] = "Search Results for \u201C" + WPStripAllTags(q, false) + "\u201D"
		} else {
			parts["title"] = "Search Results"
		}
	case s != nil && s.IsFrontPage:
		parts["title"] = BlogInfo(c, "name", "display")
	case s != nil && s.IsHome:
		parts["title"] = singularOrPostsTitle(c)
	case s != nil && (s.IsSingular || s.IsSingle || s.IsPage):
		parts["title"] = singularOrPostsTitle(c)
	case s != nil && s.IsCategory:
		parts["title"] = archiveTermTitle(c, "category")
	case s != nil && s.IsTag:
		parts["title"] = archiveTermTitle(c, "post_tag")
	case s != nil && s.IsAuthor:
		parts["title"] = authorArchiveTitle(c)
	case s != nil && s.IsDate:
		parts["title"] = dateArchiveTitle(c)
	default:
		if s != nil && s.IsHome {
			parts["title"] = singularOrPostsTitle(c)
		}
	}

	// Page number when paginated (not 404).
	if s != nil && !s.Is404 {
		paged := s.Paged
		if paged < 1 {
			paged = 1
		}
		page := s.Page
		if page < 1 {
			page = 1
		}
		if paged >= 2 || page >= 2 {
			n := paged
			if page > n {
				n = page
			}
			parts["page"] = fmt.Sprintf("Page %d", n)
		}
	}

	if s != nil && s.IsFrontPage {
		parts["tagline"] = BlogInfo(c, "description", "display")
	} else {
		parts["site"] = BlogInfo(c, "name", "display")
	}

	sep := "-"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("document_title_separator", sep); v != nil {
			if s, ok := v.(string); ok {
				sep = s
			}
		}
		if v := c.Hooks.ApplyFilters("document_title_parts", parts); v != nil {
			switch t := v.(type) {
			case map[string]string:
				parts = t
			case map[string]any:
				parts = map[string]string{}
				for k, val := range t {
					parts[k] = fmt.Sprint(val)
				}
			}
		}
	}

	ordered := make([]string, 0, 4)
	for _, key := range []string{"title", "page", "tagline", "site"} {
		if v := strings.TrimSpace(parts[key]); v != "" {
			ordered = append(ordered, v)
		}
	}
	title := strings.Join(ordered, " "+sep+" ")

	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("document_title", title); v != nil {
			if s, ok := v.(string); ok {
				title = s
			}
		}
	}
	return title
}

// RenderTitleTag returns <title>…</title>\n when the theme supports title-tag
// (_wp_render_title_tag, P3.126). Empty when unsupported.
func RenderTitleTag(c *Context) string {
	if c == nil || !CurrentThemeSupports(c, "title-tag") {
		return ""
	}
	return "<title>" + GetDocumentTitle(c) + "</title>\n"
}

func singularOrPostsTitle(c *Context) string {
	if title := queriedPostTitle(c); title != "" {
		return title
	}
	return BlogInfo(c, "name", "display")
}

func archiveTermTitle(c *Context, taxonomy string) string {
	if c == nil || c.Scope == nil {
		return ""
	}
	switch taxonomy {
	case "category":
		if c.Scope.CategorySlug != "" {
			return c.Scope.CategorySlug
		}
	case "post_tag":
		if c.Scope.TagSlug != "" {
			return c.Scope.TagSlug
		}
	}
	return ""
}

func authorArchiveTitle(c *Context) string {
	if c == nil || c.Scope == nil {
		return ""
	}
	if c.Scope.AuthorName != "" {
		return c.Scope.AuthorName
	}
	if c.Scope.AuthorID > 0 {
		if u, err := c.GetUserBy("id", c.Scope.AuthorID); err == nil && u != nil {
			if u.DisplayName != "" {
				return u.DisplayName
			}
			return u.Login
		}
	}
	return ""
}

func dateArchiveTitle(c *Context) string {
	if c == nil || c.Scope == nil {
		return ""
	}
	s := c.Scope
	if s.DateDay > 0 && s.DateMonth > 0 && s.DateYear > 0 {
		return fmt.Sprintf("%s %d, %d", monthName(s.DateMonth), s.DateDay, s.DateYear)
	}
	if s.DateMonth > 0 && s.DateYear > 0 {
		return monthName(s.DateMonth) + " " + strconv.Itoa(s.DateYear)
	}
	if s.DateYear > 0 {
		return strconv.Itoa(s.DateYear)
	}
	return ""
}
