// Navigation helpers for block rendering (P5.7).
package wp

import (
	"fmt"
	"strings"
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

// NavPage is a published page used as a navigation fallback link.
type NavPage struct {
	ID    int64
	Title string
	Name  string
}

// ListPublishedPages returns published pages ordered by title (navigation fallback).
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
	})
	if err != nil {
		return nil, err
	}
	out := make([]NavPage, 0, len(posts))
	for _, p := range posts {
		out = append(out, NavPage{ID: p.ID, Title: p.Title, Name: p.Name})
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

func optionStringValue(c *Context, name, fallback string) string {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case string:
		if t == "" {
			return fallback
		}
		return t
	default:
		s := fmt.Sprint(t)
		if s == "" || s == "<nil>" {
			return fallback
		}
		return s
	}
}
