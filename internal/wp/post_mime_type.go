package wp

import (
	"fmt"
	"strings"
)

// GetPostMimeType returns post_mime_type for a post ID (get_post_mime_type, P8.52).
// ok is false when the post cannot be loaded (WP returns false).
func GetPostMimeType(c *Context, postID int64) (mime string, ok bool) {
	if c == nil || c.DB == nil || postID <= 0 {
		return "", false
	}
	q, args, err := c.DB.Prepare(
		"SELECT post_mime_type FROM "+c.DB.Posts()+" WHERE ID = %d LIMIT 1",
		postID,
	)
	if err != nil {
		return "", false
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return "", false
	}
	mime = strings.TrimSpace(fmt.Sprint(row["post_mime_type"]))
	if mime == "<nil>" {
		mime = ""
	}
	return mime, true
}
