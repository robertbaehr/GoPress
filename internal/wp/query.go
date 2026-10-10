// Package wp — WP_Query-like post helpers (P3.6).
package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// QueryArgs is a minimal subset of WP_Query / get_posts arguments.
type QueryArgs struct {
	PostType     string // default "post"
	PostStatus   string // default "publish"
	PostsPerPage int    // default 10; -1 means no LIMIT
	Offset       int    // SQL OFFSET (ignored when PostsPerPage < 0)
	P            int64  // post ID (query var "p")
	Name         string // post slug (query var "name")
	PageName     string // page slug (query var "pagename"); implies post_type=page when PostType unset
}

// Post is a row from wp_posts used by front and REST helpers.
type Post struct {
	ID      int64
	Title   string
	Content string
	Excerpt string
	Name    string
	Status  string
	Type    string
	Date    string
}

// GetPosts runs a WP_Query-like SELECT against wp_posts.
func GetPosts(c *Context, args QueryArgs) ([]Post, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: GetPosts: nil DB")
	}

	postType := args.PostType
	if postType == "" {
		if args.PageName != "" {
			postType = "page"
		} else {
			postType = "post"
		}
	}
	status := args.PostStatus
	if status == "" {
		status = "publish"
	}
	limit := args.PostsPerPage
	if limit == 0 {
		limit = 10
	}

	var where []string
	var vals []any

	where = append(where, "post_type = %s")
	vals = append(vals, postType)
	where = append(where, "post_status = %s")
	vals = append(vals, status)

	if args.P > 0 {
		where = append(where, "ID = %d")
		vals = append(vals, args.P)
	}
	slug := args.Name
	if slug == "" {
		slug = args.PageName
	}
	if slug != "" {
		slug = lastPathSegment(slug)
		where = append(where, "post_name = %s")
		vals = append(vals, slug)
	}

	q := "SELECT ID, post_date, post_name, post_status, post_type, post_title, post_content, post_excerpt FROM " +
		c.DB.Posts() + " WHERE " + strings.Join(where, " AND ") +
		" ORDER BY post_date DESC"
	if limit >= 0 {
		q += " LIMIT %d"
		vals = append(vals, limit)
		if args.Offset > 0 {
			q += " OFFSET %d"
			vals = append(vals, args.Offset)
		}
	}

	sqlStr, sqlArgs, err := c.DB.Prepare(q, vals...)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(sqlStr, sqlArgs...)
	if err != nil {
		return nil, err
	}
	out := make([]Post, 0, len(rows))
	for _, row := range rows {
		out = append(out, postFromRow(row))
	}
	return out, nil
}

// GetPost returns the first post matching args, or nil if none.
func GetPost(c *Context, args QueryArgs) (*Post, error) {
	if args.PostsPerPage == 0 {
		args.PostsPerPage = 1
	}
	posts, err := GetPosts(c, args)
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, nil
	}
	return &posts[0], nil
}

func postFromRow(row map[string]any) Post {
	return Post{
		ID:      rowInt64(row["ID"]),
		Date:    rowString(row["post_date"]),
		Name:    rowString(row["post_name"]),
		Status:  rowString(row["post_status"]),
		Type:    rowString(row["post_type"]),
		Title:   rowString(row["post_title"]),
		Content: rowString(row["post_content"]),
		Excerpt: rowString(row["post_excerpt"]),
	}
}

func lastPathSegment(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func rowString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func rowInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case uint64:
		return int64(t)
	case []byte:
		n, _ := strconv.ParseInt(string(t), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return n
	}
}
