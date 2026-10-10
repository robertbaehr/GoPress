package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

type commentWriteBody struct {
	Post        int64  `json:"post"`
	Parent      int64  `json:"parent"`
	Content     any    `json:"content"`
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
	AuthorURL   string `json:"author_url"`
	AuthorIP    string `json:"author_ip"`
	Status      string `json:"status"`
}

func (s *Server) serveComments(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	switch r.Method {
	case http.MethodGet:
		s.listComments(w, r, c)
	case http.MethodPost:
		s.createComment(w, r, c)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) listComments(w http.ResponseWriter, r *http.Request, c *wp.Context) {
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
	if c != nil && c.CurrentUser != nil {
		w.Header().Set("Allow", "GET, POST")
	}
	writeJSON(w, http.StatusOK, comments)
}

// createComment handles POST /wp/v2/comments for logged-in users (P4.45).
func (s *Server) createComment(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_comment_login_required",
			"message": "Sorry, you must be logged in to comment.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	body, err := decodeCommentWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	if body.Post <= 0 {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_comment_invalid_post_id",
			"message": "Sorry, you are not allowed to create this comment without a post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	post, err := loadCommentablePost(c, body.Post)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_comment_invalid_post_id",
			"message": "Sorry, you are not allowed to create this comment without a post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	if strings.EqualFold(post.Status, "draft") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_comment_draft_post",
			"message": "Sorry, you are not allowed to create a comment on this post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	if strings.EqualFold(post.Status, "trash") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_comment_trash_post",
			"message": "Sorry, you are not allowed to create a comment on this post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	if strings.EqualFold(post.CommentStatus, "closed") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_comment_closed",
			"message": "Sorry, comments are closed for this item.",
			"data":    map[string]any{"status": 403},
		})
		return
	}

	content := commentContentFromBody(body.Content)
	content = strings.TrimSpace(wp.KsesData(content))
	if content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_comment_content_invalid",
			"message": "Invalid comment content.",
			"data":    map[string]any{"status": 400},
		})
		return
	}

	author := strings.TrimSpace(body.AuthorName)
	email := strings.TrimSpace(body.AuthorEmail)
	url := strings.TrimSpace(body.AuthorURL)
	if author == "" {
		author = c.CurrentUser.DisplayName
		if author == "" {
			author = c.CurrentUser.Login
		}
	}
	if email == "" {
		email = c.CurrentUser.Email
	}

	approved := wp.ResolveCommentApproval(c, email, c.CurrentUser)
	if body.Status != "" && c.UserCan(c.CurrentUser, "moderate_comments") {
		switch strings.ToLower(strings.TrimSpace(body.Status)) {
		case "approved", "approve", "1":
			approved = "1"
		case "hold", "0":
			approved = "0"
		case "spam":
			approved = "spam"
		}
	}

	parent := body.Parent
	if parent < 0 {
		parent = 0
	}
	authorIP := strings.TrimSpace(body.AuthorIP)
	if authorIP == "" {
		authorIP = requestRemoteIP(r)
	}

	id, err := wp.InsertComment(c, wp.CommentInsert{
		PostID:      body.Post,
		Author:      author,
		AuthorEmail: email,
		AuthorURL:   url,
		AuthorIP:    authorIP,
		Content:     content,
		Parent:      parent,
		UserID:      c.CurrentUser.ID,
		Agent:       truncateStr(r.UserAgent(), 255),
		Approved:    approved,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_failed",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}

	cm, err := getCommentByIDAny(c, id)
	if err != nil || cm == nil {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, cm)
}

func decodeCommentWriteBody(r *http.Request) (commentWriteBody, error) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return commentWriteBody{}, err
	}
	var body commentWriteBody
	if len(raw) == 0 {
		return body, nil
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return commentWriteBody{}, err
	}
	return body, nil
}

func commentContentFromBody(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if raw, ok := t["raw"].(string); ok {
			return raw
		}
		if rendered, ok := t["rendered"].(string); ok {
			return rendered
		}
	}
	return ""
}

func loadCommentablePost(c *wp.Context, id int64) (*wp.Post, error) {
	post, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "post", PostStatus: "publish"})
	if err != nil {
		return nil, err
	}
	if post != nil {
		return post, nil
	}
	return wp.GetPost(c, wp.QueryArgs{P: id, PostType: "page", PostStatus: "publish"})
}

// getCommentByIDAny loads a comment regardless of approval (for create responses).
func getCommentByIDAny(c *wp.Context, id int64) (*Comment, error) {
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
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	cm := commentFromRow(c, row, home)
	return &cm, nil
}

func requestRemoteIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		return host[:i]
	}
	return host
}

func truncateStr(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}
