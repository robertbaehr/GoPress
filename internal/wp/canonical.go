package wp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GetCanonicalURL returns the canonical URL for a published post
// (wp_get_canonical_url subset, P3.42 / P3.101 / P3.105 / P3.129). page/cpage
// mirror query vars for multipage content and comment pagination when > 1 / > 0
// respectively. Applies the get_canonical_url filter.
func GetCanonicalURL(c *Context, postID int64, page, cpage int) string {
	if postID <= 0 {
		return ""
	}
	post, err := GetPostByID(c, postID)
	if err != nil || post == nil {
		return ""
	}
	// P3.129: resolve inherit like get_post_status (parent status for attachments).
	if !canonicalPostIsPublic(c, post) {
		return ""
	}
	url := postCanonicalPermalink(c, post)
	if url == "" {
		return ""
	}
	structure := ""
	if c != nil && c.Options != nil {
		structure = strings.TrimSpace(fmt.Sprint(c.Options.GetOption("permalink_structure", "")))
	}
	if page >= 2 {
		if structure == "" {
			url = addQueryArg(url, "page", strconv.Itoa(page))
		} else {
			url = strings.TrimRight(url, "/") + "/" + strconv.Itoa(page) + "/"
		}
	}
	if cpage > 0 {
		// P3.129 / get_comments_pagenum_link: pretty comment-page-N + #comments.
		url = canonicalCommentsPagenumURL(url, cpage, structure != "")
	}
	// P3.101: get_canonical_url filter (url, post).
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("get_canonical_url", url, post)
		if s, ok := filtered.(string); ok {
			url = s
		}
	}
	return url
}

// canonicalPostIsPublic mirrors get_post_status()==publish for canonical (P3.129).
func canonicalPostIsPublic(c *Context, post *Post) bool {
	if post == nil {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(post.Status))
	if status == "publish" {
		return true
	}
	if status != "inherit" {
		return false
	}
	// Walk parent chain for attachment/revision inherit.
	parentID := post.Parent
	for parentID > 0 {
		parent, err := GetPostByID(c, parentID)
		if err != nil || parent == nil {
			return false
		}
		ps := strings.ToLower(strings.TrimSpace(parent.Status))
		if ps == "publish" {
			return true
		}
		if ps != "inherit" {
			return false
		}
		parentID = parent.Parent
	}
	return false
}

// canonicalCommentsPagenumURL builds get_comments_pagenum_link-style URL (P3.129).
func canonicalCommentsPagenumURL(permalink string, cpage int, pretty bool) string {
	permalink = strings.TrimSpace(permalink)
	if permalink == "" || cpage <= 0 {
		return permalink
	}
	if pretty {
		base := strings.TrimRight(permalink, "/")
		return base + "/comment-page-" + strconv.Itoa(cpage) + "/#comments"
	}
	return addQueryArg(permalink, "cpage", strconv.Itoa(cpage)) + "#comments"
}

// RelCanonicalLink returns <link rel="canonical" …> HTML for a post, or "".
func RelCanonicalLink(c *Context, postID int64) string {
	return RelCanonicalLinkPaged(c, postID, 0, 0)
}

// RelCanonicalLinkPaged is rel_canonical with page/cpage (P3.101 / P3.105).
// Href is passed through EscURL like PHP esc_url().
func RelCanonicalLinkPaged(c *Context, postID int64, page, cpage int) string {
	url := GetCanonicalURL(c, postID, page, cpage)
	if url == "" {
		return ""
	}
	return `<link rel="canonical" href="` + EscURL(url) + `" />` + "\n"
}

// CanonicalPageFromRequest reads page/cpage for singular canonical (P3.101 / P3.105 / P3.129).
// Prefers rewrite Scope.Page (pretty /N/), then request query vars, then
// /comment-page-N/ path segments (P3.129).
func CanonicalPageFromRequest(c *Context) (page, cpage int) {
	if c != nil && c.Scope != nil {
		c.Scope.mu.Lock()
		page = c.Scope.Page
		c.Scope.mu.Unlock()
	}
	if c == nil || c.Req == nil {
		return page, 0
	}
	q := c.Req.URL.Query()
	if page == 0 {
		if v := strings.TrimSpace(q.Get("page")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}
	}
	if v := strings.TrimSpace(q.Get("cpage")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cpage = n
		}
	}
	// P3.129: pretty comments pagination base comment-page-N.
	if cpage == 0 {
		path := c.Req.URL.Path
		const marker = "/comment-page-"
		if i := strings.LastIndex(path, marker); i >= 0 {
			rest := path[i+len(marker):]
			rest = strings.Trim(rest, "/")
			if n, err := strconv.Atoi(rest); err == nil && n > 0 {
				cpage = n
			}
		}
	}
	return page, cpage
}

func postCanonicalPermalink(c *Context, post *Post) string {
	home := strings.TrimRight(HomeURL(c, "/", ""), "/")
	if home == "" {
		return ""
	}
	if post.Type == "attachment" {
		if post.Name != "" {
			return home + "/" + post.Name + "/"
		}
		return fmt.Sprintf("%s/?attachment_id=%d", home, post.ID)
	}
	if post.Type == "page" && post.Name != "" {
		show := ""
		front := ""
		if c != nil && c.Options != nil {
			show = optionValueString(c.Options.GetOption("show_on_front", "posts"))
			front = optionValueString(c.Options.GetOption("page_on_front", "0"))
		}
		frontID, _ := strconv.ParseInt(strings.TrimSpace(front), 10, 64)
		if show == "page" && frontID == post.ID {
			return home + "/"
		}
		return home + "/" + post.Name + "/"
	}
	if post.Type == "post" && post.Name != "" && post.Date != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", post.Date); err == nil {
			return fmt.Sprintf("%s/%04d/%02d/%02d/%s/", home, t.Year(), int(t.Month()), t.Day(), post.Name)
		}
	}
	return fmt.Sprintf("%s/?p=%d", home, post.ID)
}

func addQueryArg(rawURL, key, value string) string {
	if strings.Contains(rawURL, "?") {
		return rawURL + "&" + key + "=" + value
	}
	return rawURL + "?" + key + "=" + value
}

func htmlEscapeAttr(s string) string {
	replacer := strings.NewReplacer(
		`&`, `&amp;`,
		`"`, `&quot;`,
		`<`, `&lt;`,
		`>`, `&gt;`,
	)
	return replacer.Replace(s)
}
