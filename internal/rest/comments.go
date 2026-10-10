package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Comment is a minimal wp/v2 comment object.
type Comment struct {
	ID         int64          `json:"id"`
	Post       int64          `json:"post"`
	Parent     int64          `json:"parent"`
	Author     int64          `json:"author"`
	AuthorName string         `json:"author_name"`
	AuthorURL  string         `json:"author_url"`
	Date       string         `json:"date"`
	DateGMT    string         `json:"date_gmt"`
	Content    RenderedString `json:"content"`
	Link       string         `json:"link"`
	Status     string         `json:"status"`
	Type       string         `json:"type"`
}

func (s *Server) serveComments(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	postID := parseOptionalInt64(r, "post")
	total, err := countComments(c, postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	comments, err := listCommentsPaged(c, postID, perPage, page)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, total, totalPages, page)
	writeJSON(w, http.StatusOK, comments)
}

func (s *Server) serveCommentItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	comment, err := getCommentByID(c, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if comment == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_comment_invalid_id",
			"message": "Invalid comment ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	writeJSON(w, http.StatusOK, comment)
}

func parseOptionalInt64(r *http.Request, key string) int64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func listCommentsPaged(c *wp.Context, postID int64, limit, page int) ([]Comment, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	var (
		q    string
		args []any
		err  error
	)
	base := "SELECT comment_ID, comment_post_ID, comment_author, comment_author_email, comment_author_url, " +
		"comment_date, comment_date_gmt, comment_content, comment_approved, comment_type, comment_parent, user_id " +
		"FROM " + c.DB.Comments() +
		" WHERE comment_approved = %s AND comment_type IN (%s, %s)"
	if postID > 0 {
		q, args, err = c.DB.Prepare(
			base+" AND comment_post_ID = %d ORDER BY comment_date_gmt DESC LIMIT %d OFFSET %d",
			"1", "comment", "", postID, limit, offset,
		)
	} else {
		q, args, err = c.DB.Prepare(
			base+" ORDER BY comment_date_gmt DESC LIMIT %d OFFSET %d",
			"1", "comment", "", limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]Comment, 0, len(rows))
	for _, row := range rows {
		out = append(out, commentFromRow(row, home))
	}
	return out, nil
}

func countComments(c *wp.Context, postID int64) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	var (
		q    string
		args []any
		err  error
	)
	base := "SELECT COUNT(*) AS total FROM " + c.DB.Comments() +
		" WHERE comment_approved = %s AND comment_type IN (%s, %s)"
	if postID > 0 {
		q, args, err = c.DB.Prepare(base+" AND comment_post_ID = %d", "1", "comment", "", postID)
	} else {
		q, args, err = c.DB.Prepare(base, "1", "comment", "")
	}
	if err != nil {
		return 0, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return int(asInt64(row["total"])), nil
}

func getCommentByID(c *wp.Context, id int64) (*Comment, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID, comment_post_ID, comment_author, comment_author_email, comment_author_url, "+
			"comment_date, comment_date_gmt, comment_content, comment_approved, comment_type, comment_parent, user_id "+
			"FROM "+c.DB.Comments()+
			" WHERE comment_ID = %d AND comment_approved = %s AND comment_type IN (%s, %s) LIMIT 1",
		id, "1", "comment", "",
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
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	cm := commentFromRow(row, home)
	return &cm, nil
}

func commentFromRow(row map[string]any, home string) Comment {
	id := asInt64(row["comment_ID"])
	postID := asInt64(row["comment_post_ID"])
	ctype := asString(row["comment_type"])
	if ctype == "" {
		ctype = "comment"
	}
	return Comment{
		ID:         id,
		Post:       postID,
		Parent:     asInt64(row["comment_parent"]),
		Author:     asInt64(row["user_id"]),
		AuthorName: asString(row["comment_author"]),
		AuthorURL:  asString(row["comment_author_url"]),
		Date:       mysqlDateToREST(asString(row["comment_date"])),
		DateGMT:    mysqlDateToREST(asString(row["comment_date_gmt"])),
		Content:    RenderedString{Rendered: asString(row["comment_content"])},
		Link:       commentPermalink(home, postID, id),
		Status:     commentStatus(asString(row["comment_approved"])),
		Type:       ctype,
	}
}

func commentStatus(approved string) string {
	switch approved {
	case "1":
		return "approved"
	case "0", "hold":
		return "hold"
	case "spam":
		return "spam"
	case "trash":
		return "trash"
	default:
		return approved
	}
}

func commentPermalink(home string, postID, commentID int64) string {
	if postID <= 0 {
		return fmt.Sprintf("%s/?p=%d#comment-%d", home, postID, commentID)
	}
	return fmt.Sprintf("%s/?p=%d#comment-%d", home, postID, commentID)
}
