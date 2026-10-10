// Navigation helpers for block rendering (P5.7) and REST fallback (P4.53).
package wp

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// LoadNavigationContent returns published wp_navigation post_content by ID.
func LoadNavigationContent(c *Context, ref int64) (string, bool) {
	if c == nil || c.DB == nil || ref <= 0 {
		return "", false
	}
	post, err := GetPost(c, QueryArgs{
		PostType:     "wp_navigation",
		P:            ref,
		PostStatus:   "publish",
		PostsPerPage: 1,
	})
	if err != nil || post == nil {
		return "", false
	}
	return post.Content, true
}

// GetMostRecentNavigation returns the newest published wp_navigation post.
func GetMostRecentNavigation(c *Context) (*Post, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: GetMostRecentNavigation: nil DB")
	}
	posts, err := GetPosts(c, QueryArgs{
		PostType:     "wp_navigation",
		PostStatus:   "publish",
		PostsPerPage: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, nil
	}
	return &posts[0], nil
}

// GetOrCreateNavigationFallback returns (and may create) a fallback Navigation
// menu post — subset of WP_Navigation_Fallback::get_fallback (P4.53).
func GetOrCreateNavigationFallback(c *Context) (*Post, error) {
	post, err := GetMostRecentNavigation(c)
	if err != nil {
		return nil, err
	}
	if post != nil {
		return post, nil
	}
	return createDefaultNavigationFallback(c)
}

func createDefaultNavigationFallback(c *Context) (*Post, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: createDefaultNavigationFallback: nil DB")
	}
	author := int64(1)
	if c.CurrentUser != nil && c.CurrentUser.ID > 0 {
		author = c.CurrentUser.ID
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	content := "<!-- wp:page-list /-->"
	title := "Navigation"
	slug := "navigation"
	home := strings.TrimRight(optionStringValue(c, "home", optionStringValue(c, "siteurl", "")), "/")
	guid := fmt.Sprintf("%s/?p=0", home)
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		author, now, now, content, title, "", "publish",
		"closed", "closed", "", slug, "", "", now, now,
		"", 0, guid, 0, "wp_navigation", "", 0,
	)
	if err != nil {
		return nil, err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	uq, uargs, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET guid = %s WHERE ID = %d",
		fmt.Sprintf("%s/?p=%d", home, id), id,
	)
	if err == nil {
		_, _ = c.DB.Exec(uq, uargs...)
	}
	return GetMostRecentNavigation(c)
}

// NavPage is a published page used as a navigation fallback link.
type NavPage struct {
	ID     int64
	Title  string
	Name   string
	Parent int64 // post_parent for nested page-list (P5.75)
}

// ListPublishedPages returns published pages ordered by menu_order, title
// (navigation fallback / core/page-list, P5.75).
func ListPublishedPages(c *Context, limit int) ([]NavPage, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: ListPublishedPages: nil DB")
	}
	if limit <= 0 {
		limit = 50
	}
	posts, err := GetPosts(c, QueryArgs{
		PostType:     "page",
		PostStatus:   "publish",
		PostsPerPage: limit,
		OrderBy:      "title",
		Order:        "ASC",
	})
	if err != nil {
		return nil, err
	}
	// WP get_pages: sort_column menu_order,post_title.
	sort.SliceStable(posts, func(i, j int) bool {
		if posts[i].MenuOrder != posts[j].MenuOrder {
			return posts[i].MenuOrder < posts[j].MenuOrder
		}
		return strings.ToLower(posts[i].Title) < strings.ToLower(posts[j].Title)
	})
	out := make([]NavPage, 0, len(posts))
	for _, p := range posts {
		out = append(out, NavPage{ID: p.ID, Title: p.Title, Name: p.Name, Parent: p.Parent})
	}
	return out, nil
}

// PagePermalink builds a simple pretty or query permalink for a page.
func PagePermalink(c *Context, page NavPage) string {
	home := strings.TrimRight(optionStringValue(c, "home", optionStringValue(c, "siteurl", "")), "/")
	if page.Name != "" {
		return home + "/" + page.Name + "/"
	}
	return fmt.Sprintf("%s/?page_id=%d", home, page.ID)
}
