// Front-end comment helpers (P5.8) — list approved comments and insert new ones.
package wp

import (
	"fmt"
	"strings"
	"time"
)

// Comment is a row from wp_comments used by front and post handlers.
type Comment struct {
	ID          int64
	PostID      int64
	Author      string
	AuthorEmail string
	AuthorURL   string
	Content     string
	Date        string
	DateGMT     string
	Approved    string
	Type        string
	Parent      int64
	UserID      int64
}

// ListApprovedComments returns approved comments for a post (oldest first).
func ListApprovedComments(c *Context, postID int64) ([]Comment, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: ListApprovedComments: nil DB")
	}
	if postID <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID, comment_post_ID, comment_author, comment_author_email, comment_author_url, "+
			"comment_date, comment_date_gmt, comment_content, comment_approved, comment_type, comment_parent, user_id "+
			"FROM "+c.DB.Comments()+
			" WHERE comment_post_ID = %d AND comment_approved = %s AND comment_type IN (%s, %s) "+
			"ORDER BY comment_date_gmt ASC",
		postID, "1", "comment", "",
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(rows))
	for _, row := range rows {
		out = append(out, commentFromRow(row))
	}
	return out, nil
}

// ListRecentComments returns approved site comments newest-first (comments feed, P3.27).
// Includes pingbacks/trackbacks (any non-empty or empty comment_type with approved=1).
func ListRecentComments(c *Context, limit int) ([]Comment, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: ListRecentComments: nil DB")
	}
	if limit <= 0 {
		limit = 10
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID, comment_post_ID, comment_author, comment_author_email, comment_author_url, "+
			"comment_date, comment_date_gmt, comment_content, comment_approved, comment_type, comment_parent, user_id "+
			"FROM "+c.DB.Comments()+
			" WHERE comment_approved = %s "+
			"ORDER BY comment_date_gmt DESC LIMIT %d",
		"1", limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(rows))
	for _, row := range rows {
		out = append(out, commentFromRow(row))
	}
	return out, nil
}

// CommentInsert holds fields for inserting a new comment.
type CommentInsert struct {
	PostID      int64
	Author      string
	AuthorEmail string
	AuthorURL   string
	AuthorIP    string
	Content     string
	Parent      int64
	UserID      int64
	Agent       string
	Approved    string // "1" or "0"
}

// CommentUpdate holds fields for updating an existing comment (P4.64).
// Zero-value / nil pointer fields are left unchanged.
type CommentUpdate struct {
	Content     *string
	Author      *string
	AuthorEmail *string
	AuthorURL   *string
	Parent      *int64
	PostID      *int64
	Approved    *string // "1", "0", "spam", "trash"
}

// UpdateComment updates a comment row (wp_update_comment subset, P4.64).
func UpdateComment(c *Context, id int64, in CommentUpdate) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("wp: UpdateComment: invalid input")
	}
	existing, err := GetCommentByID(c, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("wp: UpdateComment: not found")
	}
	content := existing.Content
	author := existing.Author
	email := existing.AuthorEmail
	url := existing.AuthorURL
	parent := existing.Parent
	postID := existing.PostID
	approved := existing.Approved
	if in.Content != nil {
		content = *in.Content
	}
	if in.Author != nil {
		author = *in.Author
	}
	if in.AuthorEmail != nil {
		email = *in.AuthorEmail
	}
	if in.AuthorURL != nil {
		url = *in.AuthorURL
	}
	if in.Parent != nil {
		parent = *in.Parent
		if parent < 0 {
			parent = 0
		}
	}
	if in.PostID != nil {
		postID = *in.PostID
	}
	if in.Approved != nil {
		approved = *in.Approved
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Comments()+" SET comment_content = %s, comment_author = %s, comment_author_email = %s, "+
			"comment_author_url = %s, comment_parent = %d, comment_post_ID = %d, comment_approved = %s "+
			"WHERE comment_ID = %d",
		content, author, email, url, parent, postID, approved, id,
	)
	if err != nil {
		return err
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		return err
	}
	wasApproved := existing.Approved == "1"
	nowApproved := approved == "1"
	if wasApproved != nowApproved {
		delta := 1
		if wasApproved {
			delta = -1
		}
		_ = bumpCommentCount(c, existing.PostID, delta)
		if postID != existing.PostID && nowApproved {
			_ = bumpCommentCount(c, postID, 1)
			_ = bumpCommentCount(c, existing.PostID, -1)
		}
	} else if postID != existing.PostID && nowApproved {
		_ = bumpCommentCount(c, existing.PostID, -1)
		_ = bumpCommentCount(c, postID, 1)
	}
	return nil
}

// GetCommentByID loads a comment by ID regardless of approval status.
func GetCommentByID(c *Context, id int64) (*Comment, error) {
	if c == nil || c.DB == nil || id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID, comment_post_ID, comment_author, comment_author_email, comment_author_url, "+
			"comment_date, comment_date_gmt, comment_content, comment_approved, comment_type, comment_parent, user_id "+
			"FROM "+c.DB.Comments()+
			" WHERE comment_ID = %d AND comment_type IN (%s, %s) LIMIT 1",
		id, "comment", "",
	)
	if err != nil {
		return nil, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	cm := commentFromRow(row)
	return &cm, nil
}

// InsertComment inserts a comment row and bumps comment_count when approved.
func InsertComment(c *Context, in CommentInsert) (int64, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("wp: InsertComment: nil DB")
	}
	if in.PostID <= 0 || strings.TrimSpace(in.Content) == "" {
		return 0, fmt.Errorf("wp: InsertComment: invalid input")
	}
	approved := in.Approved
	if approved == "" {
		approved = "0"
	}
	ctype := "comment"
	now := time.Now().UTC()
	local := now.Format("2006-01-02 15:04:05")
	gmt := now.Format("2006-01-02 15:04:05")

	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Comments()+
			" (comment_post_ID, comment_author, comment_author_email, comment_author_url, comment_author_IP, "+
			"comment_date, comment_date_gmt, comment_content, comment_karma, comment_approved, comment_agent, "+
			"comment_type, comment_parent, user_id) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %d, %s, %s, %s, %d, %d)",
		in.PostID, in.Author, in.AuthorEmail, in.AuthorURL, in.AuthorIP,
		local, gmt, in.Content, 0, approved, in.Agent,
		ctype, in.Parent, in.UserID,
	)
	if err != nil {
		return 0, err
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if approved == "1" {
		_ = bumpCommentCount(c, in.PostID, 1)
	}
	return id, nil
}

// ResolveCommentApproval returns "1" (approved) or "0" (hold) from site options.
func ResolveCommentApproval(c *Context, authorEmail string, user *User) string {
	if commentOptionTruthy(c, "comment_moderation") {
		return "0"
	}
	if user != nil {
		return "1"
	}
	if commentOptionTruthy(c, "comment_previously_approved") {
		if authorEmail != "" && emailHasApprovedComment(c, authorEmail) {
			return "1"
		}
		return "0"
	}
	return "1"
}

func emailHasApprovedComment(c *Context, email string) bool {
	if c == nil || c.DB == nil || email == "" {
		return false
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID FROM "+c.DB.Comments()+
			" WHERE comment_author_email = %s AND comment_approved = %s LIMIT 1",
		email, "1",
	)
	if err != nil {
		return false
	}
	row, err := c.DB.GetRow(q, args...)
	return err == nil && row != nil
}

func bumpCommentCount(c *Context, postID int64, delta int) error {
	if c == nil || c.DB == nil || postID <= 0 || delta == 0 {
		return nil
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET comment_count = comment_count + %d WHERE ID = %d",
		delta, postID,
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func commentOptionTruthy(c *Context, name string) bool {
	if c == nil || c.Options == nil {
		return false
	}
	return optionTruthy(c.Options.GetOption(name, ""))
}

func commentFromRow(row map[string]any) Comment {
	ctype := rowString(row["comment_type"])
	if ctype == "" {
		ctype = "comment"
	}
	return Comment{
		ID:          rowInt64(row["comment_ID"]),
		PostID:      rowInt64(row["comment_post_ID"]),
		Author:      rowString(row["comment_author"]),
		AuthorEmail: rowString(row["comment_author_email"]),
		AuthorURL:   rowString(row["comment_author_url"]),
		Content:     rowString(row["comment_content"]),
		Date:        rowString(row["comment_date"]),
		DateGMT:     rowString(row["comment_date_gmt"]),
		Approved:    rowString(row["comment_approved"]),
		Type:        ctype,
		Parent:      rowInt64(row["comment_parent"]),
		UserID:      rowInt64(row["user_id"]),
	}
}
