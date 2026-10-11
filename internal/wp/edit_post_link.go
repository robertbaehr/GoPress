package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// GetEditPostLink builds get_edit_post_link for post/page/attachment (P3.155).
// context "display" (default) uses &amp;action=edit; other values use &action=edit.
// Requires current user can edit_post. Applies get_edit_post_link filter.
func GetEditPostLink(c *Context, postID int64, context string) string {
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
		// supported
	default:
		return ""
	}
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "edit_post", postID) {
		return ""
	}
	context = strings.TrimSpace(context)
	if context == "" {
		context = "display"
	}
	action := "&action=edit"
	if context == "display" {
		action = "&amp;action=edit"
	}
	link := AdminURL(c, fmt.Sprintf("post.php?post=%s%s", strconv.FormatInt(postID, 10), action))
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_edit_post_link", link, postID, context); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return link
}
