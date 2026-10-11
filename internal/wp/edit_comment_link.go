package wp

import (
	"strconv"
	"strings"
)

// GetEditCommentLink builds get_edit_comment_link (P3.158).
// context "display" (default) uses &amp;c=; other values use &c=.
// Requires current user can edit_comment. Applies get_edit_comment_link filter.
func GetEditCommentLink(c *Context, commentID int64, context string) string {
	if c == nil || commentID <= 0 {
		return ""
	}
	cm, err := GetCommentByID(c, commentID)
	if err != nil || cm == nil {
		return ""
	}
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "edit_comment", cm.ID) {
		return ""
	}
	context = strings.TrimSpace(context)
	if context == "" {
		context = "display"
	}
	action := "comment.php?action=editcomment&c="
	if context == "display" {
		action = "comment.php?action=editcomment&amp;c="
	}
	link := AdminURL(c, action) + strconv.FormatInt(cm.ID, 10)
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_edit_comment_link", link, cm.ID, context); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return link
}
