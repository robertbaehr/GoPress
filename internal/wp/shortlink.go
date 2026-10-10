package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// GetShortlink returns a ?p= shortlink for a post (wp_get_shortlink subset, P3.40).
// context "post" uses id; context "query" uses queriedID when id is 0.
// Static front pages return the home URL.
func GetShortlink(c *Context, id int64, context string, queriedID int64) string {
	postID := id
	ctx := strings.ToLower(strings.TrimSpace(context))
	if ctx == "" {
		ctx = "post"
	}
	if ctx == "query" && postID <= 0 {
		postID = queriedID
	}
	if postID <= 0 {
		return ""
	}
	post, err := GetPostByID(c, postID)
	if err != nil || post == nil {
		return ""
	}
	switch post.Type {
	case "post", "page", "attachment":
		// public shortlink types
	default:
		return ""
	}
	if post.Type == "page" && c != nil && c.Options != nil {
		show := optionValueString(c.Options.GetOption("show_on_front", "posts"))
		front := optionValueString(c.Options.GetOption("page_on_front", "0"))
		frontID, _ := strconv.ParseInt(strings.TrimSpace(front), 10, 64)
		if show == "page" && frontID == post.ID {
			home := strings.TrimRight(HomeURL(c, "/", ""), "/")
			return home + "/"
		}
	}
	home := strings.TrimRight(HomeURL(c, "", ""), "/")
	if home == "" {
		return ""
	}
	return fmt.Sprintf("%s/?p=%d", home, post.ID)
}
