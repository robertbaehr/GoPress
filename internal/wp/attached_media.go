package wp

import "strings"

// GetAttachedMedia returns attachments of mime type attached to a post
// (get_attached_media, P8.58). type may be a mime type or prefix (e.g. "image").
// Applies get_attached_media_args and get_attached_media filters.
func GetAttachedMedia(c *Context, mimeType string, postID int64) []Post {
	if c == nil || postID <= 0 {
		return nil
	}
	post, err := GetPost(c, QueryArgs{P: postID, PostStatus: "any"})
	if err != nil || post == nil {
		return nil
	}
	args := QueryArgs{
		PostType:     "attachment",
		PostStatus:   "inherit",
		ParentIn:     []int64{post.ID},
		OrderBy:      "menu_order",
		Order:        "ASC",
		PostsPerPage: -1,
	}
	if c.Hooks != nil {
		seed := map[string]any{
			"post_parent":    post.ID,
			"post_type":      "attachment",
			"post_mime_type": mimeType,
			"posts_per_page": -1,
			"orderby":        "menu_order",
			"order":          "ASC",
		}
		if v := c.Hooks.ApplyFilters("get_attached_media_args", seed, mimeType, post); v != nil {
			if m, ok := v.(map[string]any); ok {
				if pp, ok := attachedMediaInt64(m["post_parent"]); ok && pp > 0 {
					args.ParentIn = []int64{pp}
				}
				if s, ok := m["post_mime_type"].(string); ok {
					mimeType = s
				}
				if s, ok := m["orderby"].(string); ok && s != "" {
					args.OrderBy = s
				}
				if s, ok := m["order"].(string); ok && s != "" {
					args.Order = s
				}
			}
		}
	}
	children, err := GetPosts(c, args)
	if err != nil {
		return nil
	}
	mimeType = strings.TrimSpace(mimeType)
	if mimeType != "" {
		filtered := children[:0]
		for _, ch := range children {
			mime := GetPostMimeTypeString(c, ch.ID)
			if mimeMatches(mime, mimeType) {
				filtered = append(filtered, ch)
			}
		}
		children = filtered
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_attached_media", children, mimeType, post); v != nil {
			if posts, ok := v.([]Post); ok {
				return posts
			}
		}
	}
	return children
}

func attachedMediaInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

func mimeMatches(mime, want string) bool {
	mime = strings.ToLower(strings.TrimSpace(mime))
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return true
	}
	if mime == want {
		return true
	}
	// Prefix match: "image" matches "image/jpeg".
	if !strings.Contains(want, "/") && strings.HasPrefix(mime, want+"/") {
		return true
	}
	return false
}

// GetPostMimeTypeString returns post_mime_type or "".
func GetPostMimeTypeString(c *Context, postID int64) string {
	mime, ok := GetPostMimeType(c, postID)
	if !ok {
		return ""
	}
	return mime
}
