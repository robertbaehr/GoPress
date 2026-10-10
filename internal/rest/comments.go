package rest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Comment is a wp/v2 comment object (view context).
type Comment struct {
	ID               int64             `json:"id"`
	Post             int64             `json:"post"`
	Parent           int64             `json:"parent"`
	Author           int64             `json:"author"`
	AuthorName       string            `json:"author_name"`
	AuthorURL        string            `json:"author_url"`
	Date             string            `json:"date"`
	DateGMT          string            `json:"date_gmt"`
	Content          RenderedString    `json:"content"`
	Link             string            `json:"link"`
	Status           string            `json:"status"`
	Type             string            `json:"type"`
	AuthorAvatarURLs map[string]string `json:"author_avatar_urls"`
	Meta             map[string]any    `json:"meta"`
	Links            map[string]any    `json:"_links"`
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
	setCollectionHeaders(w, r, c, total, totalPages, page)
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
		out = append(out, commentFromRow(c, row, home))
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
	cm := commentFromRow(c, row, home)
	return &cm, nil
}

func commentFromRow(c *wp.Context, row map[string]any, home string) Comment {
	id := asInt64(row["comment_ID"])
	postID := asInt64(row["comment_post_ID"])
	parent := asInt64(row["comment_parent"])
	author := asInt64(row["user_id"])
	email := asString(row["comment_author_email"])
	if author > 0 {
		if userEmail := userEmailByID(c, author); userEmail != "" {
			email = userEmail
		}
	}
	ctype := asString(row["comment_type"])
	if ctype == "" {
		ctype = "comment"
	}
	return Comment{
		ID:               id,
		Post:             postID,
		Parent:           parent,
		Author:           author,
		AuthorName:       asString(row["comment_author"]),
		AuthorURL:        asString(row["comment_author_url"]),
		Date:             mysqlDateToREST(asString(row["comment_date"])),
		DateGMT:          mysqlDateToREST(asString(row["comment_date_gmt"])),
		Content:          RenderedString{Rendered: commentContentHTML(asString(row["comment_content"]))},
		Link:             commentPermalink(c, home, postID, id),
		Status:           commentStatus(asString(row["comment_approved"])),
		Type:             ctype,
		AuthorAvatarURLs: gravatarAvatarURLs(email),
		Meta:             map[string]any{"_wp_note_status": nil},
		Links:            buildCommentLinks(c, home, id, postID, parent, author),
	}
}

func userEmailByID(c *wp.Context, userID int64) string {
	if c == nil || c.DB == nil || userID <= 0 {
		return ""
	}
	q, args, err := c.DB.Prepare("SELECT user_email FROM "+c.DB.Users()+" WHERE ID = %d LIMIT 1", userID)
	if err != nil {
		return ""
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return ""
	}
	return asString(row["user_email"])
}

func commentContentHTML(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "<p>") || strings.Contains(raw, "<p ") {
		if !strings.HasSuffix(raw, "\n") {
			raw += "\n"
		}
		return raw
	}
	// Minimal wpautop for single-block comments: wrap in <p>, newlines → <br />.
	parts := strings.Split(raw, "\n")
	var b strings.Builder
	b.WriteString("<p>")
	for i, part := range parts {
		if i > 0 {
			b.WriteString("<br />\n")
		}
		b.WriteString(part)
	}
	b.WriteString("</p>\n")
	return b.String()
}

func gravatarAvatarURLs(email string) map[string]string {
	email = strings.ToLower(strings.TrimSpace(email))
	sum := sha256.Sum256([]byte(email))
	hash := hex.EncodeToString(sum[:])
	out := map[string]string{}
	for _, size := range []int{24, 48, 96} {
		out[strconv.Itoa(size)] = fmt.Sprintf(
			"https://secure.gravatar.com/avatar/%s?s=%d&d=mm&r=g", hash, size,
		)
	}
	return out
}

func buildCommentLinks(c *wp.Context, home string, id, postID, parent, author int64) map[string]any {
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	postType := "post"
	if c != nil && c.DB != nil && postID > 0 {
		if pt := postTypeByID(c, postID); pt != "" {
			postType = pt
		}
	}
	base := "posts"
	if postType == "page" {
		base = "pages"
	}
	links := map[string]any{
		"self": []map[string]any{{
			"href":        fmt.Sprintf("%s/comments/%d", restRoot, id),
			"targetHints": map[string]any{"allow": []string{"GET"}},
		}},
		"collection": []map[string]any{{"href": restRoot + "/comments"}},
		"up": []map[string]any{{
			"embeddable": true,
			"post_type":  postType,
			"href":       fmt.Sprintf("%s/%s/%d", restRoot, base, postID),
		}},
	}
	if author > 0 {
		links["author"] = []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/users/%d", restRoot, author),
		}}
	}
	if parent > 0 {
		links["in-reply-to"] = []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/comments/%d", restRoot, parent),
		}}
	}
	if commentHasChildren(c, id) {
		links["children"] = []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/comments?parent=%d", restRoot, id),
		}}
	}
	return links
}

func postTypeByID(c *wp.Context, id int64) string {
	if c == nil || c.DB == nil || id <= 0 {
		return ""
	}
	q, args, err := c.DB.Prepare("SELECT post_type FROM "+c.DB.Posts()+" WHERE ID = %d LIMIT 1", id)
	if err != nil {
		return ""
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return ""
	}
	return asString(row["post_type"])
}

func commentHasChildren(c *wp.Context, parentID int64) bool {
	if c == nil || c.DB == nil || parentID <= 0 {
		return false
	}
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID FROM "+c.DB.Comments()+
			" WHERE comment_parent = %d AND comment_approved = %s AND comment_type IN (%s, %s) LIMIT 1",
		parentID, "1", "comment", "",
	)
	if err != nil {
		return false
	}
	row, err := c.DB.GetRow(q, args...)
	return err == nil && row != nil
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

func commentPermalink(c *wp.Context, home string, postID, commentID int64) string {
	if c != nil && c.DB != nil && postID > 0 {
		q, args, err := c.DB.Prepare(
			"SELECT ID, post_date, post_name, post_type, post_parent FROM "+c.DB.Posts()+" WHERE ID = %d LIMIT 1",
			postID,
		)
		if err == nil {
			row, err := c.DB.GetRow(q, args...)
			if err == nil && row != nil {
				p := wp.Post{
					ID:     asInt64(row["ID"]),
					Date:   asString(row["post_date"]),
					Name:   asString(row["post_name"]),
					Type:   asString(row["post_type"]),
					Parent: asInt64(row["post_parent"]),
				}
				return itemPermalink(c, home, p) + fmt.Sprintf("#comment-%d", commentID)
			}
		}
	}
	return fmt.Sprintf("%s/?p=%d#comment-%d", home, postID, commentID)
}
