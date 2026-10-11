package wp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// GetDeletePostLink builds get_delete_post_link for post/page/attachment (P3.156).
// forceDelete or EmptyTrashDays()==0 → action=delete; else action=trash.
// Requires delete_post capability. Applies get_delete_post_link filter.
func GetDeletePostLink(c *Context, postID int64, forceDelete bool) string {
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
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "delete_post", postID) {
		return ""
	}
	action := "trash"
	if forceDelete || EmptyTrashDays() == 0 {
		action = "delete"
	}
	base := AdminURL(c, fmt.Sprintf("post.php?post=%s", strconv.FormatInt(postID, 10)))
	link := base
	if u, err := url.Parse(base); err == nil {
		q := u.Query()
		q.Set("action", action)
		u.RawQuery = q.Encode()
		link = u.String()
	} else {
		sep := "?"
		if strings.Contains(base, "?") {
			sep = "&"
		}
		link = base + sep + "action=" + action
	}
	if nonce := c.CreateNonce(action + "-post_" + strconv.FormatInt(postID, 10)); nonce != "" {
		sep := "&"
		if !strings.Contains(link, "?") {
			sep = "?"
		}
		link = link + sep + "_wpnonce=" + url.QueryEscape(nonce)
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_delete_post_link", link, postID, forceDelete); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return link
}

// EmptyTrashDays returns EMPTY_TRASH_DAYS (default 30). Zero disables trash.
func EmptyTrashDays() int {
	return 30
}
