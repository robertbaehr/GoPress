package wp

import (
	"html"
	"strconv"
	"strings"
)

// CommentPaging describes front comment pagination (page_comments / cpage, P3.108).
type CommentPaging struct {
	Enabled bool
	PerPage int
	Page    int // 1-based
	Total   int // total pages (≥1 when Enabled and comments exist)
}

// CommentPagingForPost builds paging state from options + request cpage (P3.108).
// When page_comments is off, Enabled is false and callers should render all comments.
func CommentPagingForPost(c *Context, totalComments int) CommentPaging {
	out := CommentPaging{PerPage: 50, Page: 1, Total: 1}
	if c == nil || totalComments <= 0 {
		return out
	}
	if optionStringValue(c, "page_comments", "0") != "1" {
		return out
	}
	perPage := 50
	if v := strings.TrimSpace(optionStringValue(c, "comments_per_page", "50")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}
	totalPages := (totalComments + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	_, cpage := CanonicalPageFromRequest(c)
	page := cpage
	if page <= 0 {
		if strings.EqualFold(strings.TrimSpace(optionStringValue(c, "default_comments_page", "newest")), "oldest") {
			page = 1
		} else {
			page = totalPages // newest (WP default)
		}
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	return CommentPaging{Enabled: true, PerPage: perPage, Page: page, Total: totalPages}
}

// SliceCommentPage returns the comment slice for paging (P3.108).
// When paging is disabled, returns comments unchanged.
func SliceCommentPage[T any](comments []T, paging CommentPaging) []T {
	if !paging.Enabled || paging.PerPage <= 0 || len(comments) == 0 {
		return comments
	}
	start := (paging.Page - 1) * paging.PerPage
	if start >= len(comments) {
		return nil
	}
	end := start + paging.PerPage
	if end > len(comments) {
		end = len(comments)
	}
	return comments[start:end]
}

// CommentsPagenumLink builds a singular permalink with ?cpage=N (P3.108).
// Unlike archive paginate_links, page 1 keeps cpage=1 so default_comments_page=newest
// (bare permalink = last page) stays unambiguous.
func CommentsPagenumLink(permalink string, page int) string {
	permalink = strings.TrimSpace(permalink)
	if permalink == "" || page <= 0 {
		return permalink
	}
	return addQueryArg(permalink, "cpage", strconv.Itoa(page))
}

// CommentPaginationHTML emits nav.navigation.comment-navigation markup (P3.108).
func CommentPaginationHTML(permalink string, paging CommentPaging) string {
	if !paging.Enabled || paging.Total < 2 {
		return ""
	}
	var pageLinks []string
	if paging.Page > 1 {
		href := CommentsPagenumLink(permalink, paging.Page-1) + "#comments"
		pageLinks = append(pageLinks, `<a class="prev page-numbers" href="`+html.EscapeString(href)+`">&laquo; Older Comments</a>`)
	}
	for n := 1; n <= paging.Total; n++ {
		if n == paging.Page {
			pageLinks = append(pageLinks, `<span aria-current="page" class="page-numbers current">`+strconv.Itoa(n)+`</span>`)
			continue
		}
		href := CommentsPagenumLink(permalink, n) + "#comments"
		pageLinks = append(pageLinks, `<a class="page-numbers" href="`+html.EscapeString(href)+`">`+strconv.Itoa(n)+`</a>`)
	}
	if paging.Page < paging.Total {
		href := CommentsPagenumLink(permalink, paging.Page+1) + "#comments"
		pageLinks = append(pageLinks, `<a class="next page-numbers" href="`+html.EscapeString(href)+`">Newer Comments &raquo;</a>`)
	}
	var b strings.Builder
	b.WriteString(`<nav class="navigation comment-navigation" aria-label="Comments">` + "\n")
	b.WriteString(`<h2 class="screen-reader-text">Comment navigation</h2>` + "\n")
	b.WriteString(`<div class="nav-links">` + "\n")
	b.WriteString("<ul class='page-numbers'>\n\t<li>")
	b.WriteString(strings.Join(pageLinks, "</li>\n\t<li>"))
	b.WriteString("</li>\n</ul>\n")
	b.WriteString("</div>\n</nav>\n")
	return b.String()
}
