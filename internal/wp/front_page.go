// Front page display settings (show_on_front / page_on_front / page_for_posts) — P3.80.
package wp

import (
	"strconv"
	"strings"
)

// ApplyFrontPageDisplay adjusts ParseResult + Scope for Reading Settings
// "Your homepage displays" (show_on_front / page_on_front / page_for_posts).
//
// When show_on_front=page and page_on_front is set:
//   - `/` becomes the static front page (is_front_page, not is_home)
//   - the page_for_posts permalink becomes the blog index (is_home + is_posts_page)
//   - visiting page_on_front by slug also sets is_front_page
func ApplyFrontPageDisplay(c *Context, res *ParseResult) {
	if c == nil || res == nil || c.Options == nil || c.Scope == nil {
		return
	}
	show := strings.TrimSpace(optionValueString(c.Options.GetOption("show_on_front", "posts")))
	if show != "page" {
		// Latest posts on `/`: is_home and is_front_page stay aligned.
		return
	}
	frontID := optionInt64(c, "page_on_front", 0)
	postsID := optionInt64(c, "page_for_posts", 0)
	if frontID <= 0 {
		return
	}

	if res.QueryVars == nil {
		res.QueryVars = map[string]string{}
	}

	// Site root (and /page/N/ with no content vars) → static front page.
	if res.IsHome {
		res.IsHome = false
		res.QueryVars["page_id"] = strconv.FormatInt(frontID, 10)
		c.Scope.mu.Lock()
		c.Scope.IsHome = false
		c.Scope.IsFrontPage = true
		c.Scope.IsPostsPage = false
		c.Scope.IsPage = true
		c.Scope.IsSingular = true
		c.Scope.IsSingle = false
		c.Scope.QueriedObjectID = frontID
		c.Scope.mu.Unlock()
		return
	}

	pageID := queriedPageID(c, res)
	if pageID <= 0 {
		return
	}

	c.Scope.mu.Lock()
	defer c.Scope.mu.Unlock()
	if pageID == frontID {
		c.Scope.IsFrontPage = true
		c.Scope.IsHome = false
		c.Scope.IsPostsPage = false
		c.Scope.IsPage = true
		c.Scope.IsSingular = true
		c.Scope.QueriedObjectID = frontID
		return
	}
	if postsID > 0 && pageID == postsID {
		// Blog posts index on a static page.
		res.IsHome = true
		c.Scope.IsHome = true
		c.Scope.IsPostsPage = true
		c.Scope.IsFrontPage = false
		c.Scope.IsPage = true
		c.Scope.IsSingular = true
		c.Scope.QueriedObjectID = postsID
	}
}

func queriedPageID(c *Context, res *ParseResult) int64 {
	if res == nil {
		return 0
	}
	if idStr := strings.TrimSpace(res.QueryVars["page_id"]); idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil && id > 0 {
			return id
		}
	}
	pagename := strings.TrimSpace(res.QueryVars["pagename"])
	if pagename == "" {
		return 0
	}
	// Last path segment for hierarchical page paths.
	if i := strings.LastIndex(pagename, "/"); i >= 0 {
		pagename = pagename[i+1:]
	}
	if pagename == "" || c == nil {
		return 0
	}
	posts, err := GetPosts(c, QueryArgs{
		PostType:     "page",
		PostStatus:   "publish",
		PageName:     pagename,
		PostsPerPage: 1,
	})
	if err != nil || len(posts) == 0 {
		return 0
	}
	return posts[0].ID
}

func optionInt64(c *Context, name string, def int64) int64 {
	if c == nil || c.Options == nil {
		return def
	}
	raw := optionValueString(c.Options.GetOption(name, def))
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return def
	}
	return n
}
