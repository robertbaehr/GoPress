package wp

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var reArchivePagedSuffix = regexp.MustCompile(`(?i)/page/\d+/?$`)

// GetPagenumLink returns the archive URL for page N (get_pagenum_link subset, P3.43).
// page <= 1 yields the archive base (trailing slash). Pretty permalinks use /page/N/.
func GetPagenumLink(archiveBase string, page int) string {
	base := strings.TrimRight(strings.TrimSpace(archiveBase), "/") + "/"
	if page <= 1 {
		return base
	}
	return base + "page/" + strconv.Itoa(page) + "/"
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

// ArchiveRelPrevNextHTML emits <link rel="prev|next"> for paginated archives (P3.43).
// Empty when there is no previous/next page.
func ArchiveRelPrevNextHTML(archiveBase string, paged, maxPages int) string {
	if archiveBase == "" || maxPages < 2 {
		return ""
	}
	if paged < 1 {
		paged = 1
	}
	var b strings.Builder
	if paged > 1 {
		b.WriteString(`<link rel="prev" href="`)
		b.WriteString(html.EscapeString(GetPagenumLink(archiveBase, paged-1)))
		b.WriteString(`" />` + "\n")
	}
	if paged < maxPages {
		b.WriteString(`<link rel="next" href="`)
		b.WriteString(html.EscapeString(GetPagenumLink(archiveBase, paged+1)))
		b.WriteString(`" />` + "\n")
	}
	return b.String()
}
