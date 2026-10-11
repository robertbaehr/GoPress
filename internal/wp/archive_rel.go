package wp

import (
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var reArchivePagedSuffix = regexp.MustCompile(`(?i)/page/\d+/?$`)

// GetPagenumLink returns the archive URL for page N (get_pagenum_link subset, P3.43 / P3.107).
// page <= 1 yields the archive base (trailing slash). Pretty permalinks use /page/N/.
// querySuffix is an optional raw query without '?' (e.g. "s=hello") preserved on every page
// (WP search pagination: /page/2/?s=term).
func GetPagenumLink(archiveBase string, page int) string {
	return GetPagenumLinkQuery(archiveBase, page, "")
}

// GetPagenumLinkQuery is GetPagenumLink with an optional query string (P3.107).
func GetPagenumLinkQuery(archiveBase string, page int, querySuffix string) string {
	base := strings.TrimRight(strings.TrimSpace(archiveBase), "/") + "/"
	var href string
	if page <= 1 {
		href = base
	} else {
		href = base + "page/" + strconv.Itoa(page) + "/"
	}
	querySuffix = strings.TrimPrefix(strings.TrimSpace(querySuffix), "?")
	if querySuffix != "" {
		if strings.Contains(href, "?") {
			href += "&" + querySuffix
		} else {
			href += "?" + querySuffix
		}
	}
	return href
}

// GetPagenumLinkCtx is GetPagenumLinkQuery plus the get_pagenum_link filter (P3.145).
func GetPagenumLinkCtx(c *Context, archiveBase string, page int, querySuffix string) string {
	href := GetPagenumLinkQuery(archiveBase, page, querySuffix)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_pagenum_link", href, page); v != nil {
			if s, ok := v.(string); ok {
				href = s
			}
		}
	}
	return href
}

// ArchiveBaseFromRequest returns the absolute archive base URL (without /page/N/).
func ArchiveBaseFromRequest(c *Context, home string) string {
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	if home == "" {
		return ""
	}
	path := "/"
	if c != nil && c.Req != nil && c.Req.URL != nil {
		path = c.Req.URL.Path
	}
	path = reArchivePagedSuffix.ReplaceAllString(path, "")
	if path == "" {
		path = "/"
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return home + path
}

// ArchiveSearchQuerySuffix returns s=<term> for search archives (P3.107).
// Empty when not a search request or the term is empty.
func ArchiveSearchQuerySuffix(c *Context) string {
	if c == nil || c.Scope == nil || !c.Scope.IsSearch {
		return ""
	}
	term := strings.TrimSpace(c.Scope.SearchQuery)
	if term == "" && c.Req != nil && c.Req.URL != nil {
		term = strings.TrimSpace(c.Req.URL.Query().Get("s"))
	}
	if term == "" {
		return ""
	}
	return "s=" + url.QueryEscape(term)
}

// ArchiveRelPrevNextHTML emits <link rel="prev|next"> for paginated archives (P3.43).
// Empty when there is no previous/next page.
func ArchiveRelPrevNextHTML(archiveBase string, paged, maxPages int) string {
	return ArchiveRelPrevNextHTMLQuery(archiveBase, paged, maxPages, "")
}

// ArchiveRelPrevNextHTMLQuery is ArchiveRelPrevNextHTML with query suffix (P3.107).
func ArchiveRelPrevNextHTMLQuery(archiveBase string, paged, maxPages int, querySuffix string) string {
	return ArchiveRelPrevNextHTMLQueryCtx(nil, archiveBase, paged, maxPages, querySuffix)
}

// ArchiveRelPrevNextHTMLQueryCtx applies get_pagenum_link + EscURL on hrefs (P3.145).
func ArchiveRelPrevNextHTMLQueryCtx(c *Context, archiveBase string, paged, maxPages int, querySuffix string) string {
	if archiveBase == "" || maxPages < 2 {
		return ""
	}
	if paged < 1 {
		paged = 1
	}
	var b strings.Builder
	if paged > 1 {
		href := GetPagenumLinkCtx(c, archiveBase, paged-1, querySuffix)
		b.WriteString(`<link rel="prev" href="`)
		b.WriteString(html.EscapeString(EscURL(href)))
		b.WriteString(`" />` + "\n")
	}
	if paged < maxPages {
		href := GetPagenumLinkCtx(c, archiveBase, paged+1, querySuffix)
		b.WriteString(`<link rel="next" href="`)
		b.WriteString(html.EscapeString(EscURL(href)))
		b.WriteString(`" />` + "\n")
	}
	return b.String()
}
