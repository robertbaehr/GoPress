package wp

import (
	"net/url"
	"strconv"
	"strings"
)

// AttachmentURLToPostID resolves an attachment URL to its post ID
// (attachment_url_to_postid, P8.57).
// Honors pre_attachment_url_to_postid (non-nil int short-circuit, incl. 0) and
// attachment_url_to_postid filters.
func AttachmentURLToPostID(c *Context, rawURL string) int64 {
	rawURL = strings.TrimSpace(rawURL)
	if c == nil || rawURL == "" {
		return 0
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("pre_attachment_url_to_postid", nil, rawURL); v != nil {
			switch n := v.(type) {
			case int64:
				return n
			case int:
				return int64(n)
			case float64:
				return int64(n)
			case string:
				if id, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64); err == nil {
					return id
				}
			}
		}
	}
	uploads := GetUploadDir(c)
	path := rawURL
	if uploads.Baseurl != "" {
		base, err1 := url.Parse(uploads.Baseurl)
		img, err2 := url.Parse(path)
		if err1 == nil && err2 == nil && base != nil && img != nil &&
			img.Scheme != "" && base.Scheme != "" && img.Scheme != base.Scheme {
			path = strings.Replace(path, img.Scheme, base.Scheme, 1)
		}
		prefix := strings.TrimRight(uploads.Baseurl, "/") + "/"
		path = strings.TrimPrefix(path, prefix)
	}
	path = strings.TrimPrefix(path, "/")
	var postID int64
	if c.DB != nil && path != "" {
		q, args, err := c.DB.Prepare(
			"SELECT post_id, meta_value FROM "+c.DB.PostMeta()+
				" WHERE meta_key = %s AND meta_value = %s",
			"_wp_attached_file", path,
		)
		if err == nil {
			rows, err := c.DB.Query(q, args...)
			if err == nil {
				defer rows.Close()
				var first int64
				var exact int64
				var n int
				for rows.Next() {
					var id int64
					var meta string
					if err := rows.Scan(&id, &meta); err != nil {
						continue
					}
					n++
					if n == 1 {
						first = id
					}
					if meta == path {
						exact = id
					}
				}
				if n > 0 {
					postID = first
					if n > 1 && exact > 0 {
						postID = exact
					}
				}
			}
		}
	}
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("attachment_url_to_postid", postID, rawURL); v != nil {
			switch n := v.(type) {
			case int64:
				return n
			case int:
				return int64(n)
			case float64:
				return int64(n)
			}
		}
	}
	return postID
}
