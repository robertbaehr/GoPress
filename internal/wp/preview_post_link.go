package wp

import (
	"net/url"
	"strings"
)

// GetPreviewPostLink builds get_preview_post_link (P3.157).
// Uses the post permalink (or previewLink when non-empty), forces preview=true,
// merges extra queryArgs, then applies preview_post_link filter.
// Non-viewable types (outside post/page/attachment) return "".
func GetPreviewPostLink(c *Context, postID int64, queryArgs map[string]string, previewLink string) string {
	if c == nil || postID <= 0 {
		return ""
	}
	post, err := GetPost(c, QueryArgs{P: postID, PostStatus: "any"})
	if err != nil || post == nil {
		return ""
	}
	typ := strings.TrimSpace(post.Type)
	switch typ {
	case "post", "page", "attachment":
	default:
		return ""
	}
	link := strings.TrimSpace(previewLink)
	if link == "" {
		link = postCanonicalPermalink(c, post)
		link = SetURLScheme(c, link, "")
	}
	if link == "" {
		return ""
	}
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	q := u.Query()
	for k, v := range queryArgs {
		if strings.TrimSpace(k) == "" {
			continue
		}
		q.Set(k, v)
	}
	q.Set("preview", "true")
	u.RawQuery = q.Encode()
	link = u.String()
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("preview_post_link", link, post); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return link
}
