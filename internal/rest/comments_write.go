package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// commentWriteBody is the POST/PATCH body for comments (P4.45 / P4.117).
// Pointer fields are omit-safe on update so zero values (parent=0) can be written.
type commentWriteBody struct {
	Post        *int64  `json:"post"`
	Parent      *int64  `json:"parent"`
	Content     any     `json:"content"`
	AuthorName  *string `json:"author_name"`
	AuthorEmail *string `json:"author_email"`
	AuthorURL   *string `json:"author_url"`
	AuthorIP    *string `json:"author_ip"`
	Status      string  `json:"status"`
	Date        *string `json:"date"`
	DateGMT     *string `json:"date_gmt"`
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
	// P4.98: ?status=approve|hold|spam|trash|all (non-approve needs edit_posts).
	status, code, msg := parseCommentsStatus(r, c)
	if code != 0 {
		writeJSON(w, code, map[string]any{
			"code":    "rest_forbidden_param",
			"message": msg,
			"data":    map[string]any{"status": code, "params": map[string]any{"status": msg}},
		})
		return
	}
	// P4.129: collection filters orderby/order/search/parent/author/include/exclude.
	q := commentListQuery{
		PostID:  parseOptionalInt64(r, "post"),
		Status:  status,
		Search:  parseCommentsSearch(r),
		Orderby: parseCommentsOrderby(r),
		Order:   parseCommentsOrder(r),
		Author:  parseCommentsAuthorIDs(r),
		Parent:  parseCommentsParentIDs(r),
		Include: parseCommentsInclude(r),
		Exclude: parseCommentsExclude(r),
	}
	total, err := countComments(c, q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	comments, err := listCommentsPaged(c, q, perPage, page)
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
	if body.Post == nil || *body.Post <= 0 {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_comment_invalid_post_id",
			"message": "Sorry, you are not allowed to create this comment without a post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	postID := *body.Post
	post, err := loadCommentablePost(c, postID)
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

	author := ""
	if body.AuthorName != nil {
		author = strings.TrimSpace(*body.AuthorName)
	}
	email := ""
	if body.AuthorEmail != nil {
		email = strings.TrimSpace(*body.AuthorEmail)
	}
	url := ""
	if body.AuthorURL != nil {
		url = strings.TrimSpace(*body.AuthorURL)
	}
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

	parent := int64(0)
	if body.Parent != nil {
		parent = *body.Parent
		if parent < 0 {
			parent = 0
		}
	}
	authorIP := ""
	if body.AuthorIP != nil {
		authorIP = strings.TrimSpace(*body.AuthorIP)
	}
	if authorIP == "" {
		authorIP = requestRemoteIP(r)
	}

	id, err := wp.InsertComment(c, wp.CommentInsert{
		PostID:      postID,
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

// updateComment handles PUT/PATCH /wp/v2/comments/{id} for moderators (P4.64 / P4.117).
// P4.117: omit-safe parent/post (incl. parent=0), author_ip, date/date_gmt.
func (s *Server) updateComment(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this comment.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "moderate_comments") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this comment.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	existing, err := getCommentByIDAny(c, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_comment_invalid_id",
			"message": "Invalid comment ID.",
			"data":    map[string]any{"status": 404},
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

	upd := wp.CommentUpdate{}
	if body.Content != nil {
		content := strings.TrimSpace(wp.KsesData(commentContentFromBody(body.Content)))
		if content == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_comment_content_invalid",
				"message": "Invalid comment content.",
				"data":    map[string]any{"status": 400},
			})
			return
		}
		upd.Content = &content
	}
	if body.AuthorName != nil {
		v := strings.TrimSpace(*body.AuthorName)
		upd.Author = &v
	}
	if body.AuthorEmail != nil {
		v := strings.TrimSpace(*body.AuthorEmail)
		upd.AuthorEmail = &v
	}
	if body.AuthorURL != nil {
		v := strings.TrimSpace(*body.AuthorURL)
		upd.AuthorURL = &v
	}
	if body.AuthorIP != nil {
		v := strings.TrimSpace(*body.AuthorIP)
		upd.AuthorIP = &v
	}
	if body.Parent != nil {
		p := *body.Parent
		if p < 0 {
			p = 0
		}
		upd.Parent = &p
	}
	if body.Post != nil {
		if *body.Post <= 0 {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"code":    "rest_comment_invalid_post_id",
				"message": "Invalid post ID.",
				"data":    map[string]any{"status": 403},
			})
			return
		}
		post, err := loadCommentablePost(c, *body.Post)
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
				"message": "Invalid post ID.",
				"data":    map[string]any{"status": 403},
			})
			return
		}
		pid := *body.Post
		upd.PostID = &pid
	}
	if body.Status != "" {
		approved := commentStatusToApproved(body.Status)
		upd.Approved = &approved
	}
	if body.Date != nil {
		if mysql := restDateToMySQL(*body.Date); mysql != "" {
			upd.Date = &mysql
		}
	}
	if body.DateGMT != nil {
		if mysql := restDateToMySQL(*body.DateGMT); mysql != "" {
			upd.DateGMT = &mysql
		}
	}

	if err := wp.UpdateComment(c, id, upd); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_failed_edit",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}

	cm, err := getCommentByIDAny(c, id)
	if err != nil || cm == nil {
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, cm)
}

// deleteComment handles DELETE /wp/v2/comments/{id} (P4.80).
// force=true hard-deletes; otherwise trash (comment_approved=trash).
func (s *Server) deleteComment(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete this comment.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "moderate_comments") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete this comment.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	existing, err := getCommentByIDAny(c, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_comment_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_comment_invalid_id",
			"message": "Invalid comment ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}

	force := r.URL.Query().Get("force") == "true"
	if force {
		if err := wp.DeleteComment(c, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_cannot_delete",
				"message": "The comment cannot be deleted.",
				"data":    map[string]any{"status": 500},
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted":  true,
			"previous": existing,
		})
		return
	}

	if existing.Status == "trash" {
		writeJSON(w, http.StatusGone, map[string]any{
			"code":    "rest_already_trashed",
			"message": "The comment has already been trashed.",
			"data":    map[string]any{"status": 410},
		})
		return
	}
	if err := wp.TrashComment(c, id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "The comment cannot be deleted.",
			"data":    map[string]any{"status": 500},
		})
		return
	}
	cm, err := getCommentByIDAny(c, id)
	if err != nil || cm == nil {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "trash"})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, cm)
}

func commentStatusToApproved(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "approve", "1":
		return "1"
	case "hold", "0":
		return "0"
	case "spam":
		return "spam"
	case "trash":
		return "trash"
	default:
		return "0"
	}
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

// restDateToMySQL converts REST ISO8601 / MySQL datetime to MySQL form (P4.117).
func restDateToMySQL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format("2006-01-02 15:04:05")
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return ""
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
		return "127.0.0.1"
	}
	raw := ""
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		raw = strings.TrimSpace(parts[0])
	} else {
		host := r.RemoteAddr
		if i := strings.LastIndex(host, ":"); i >= 0 {
			// Prefer splitting host:port; keep IPv6 [::1]:port shape for RestIsIPAddress.
			if strings.HasPrefix(host, "[") {
				if end := strings.Index(host, "]"); end > 0 {
					raw = host[1:end]
				} else {
					raw = host
				}
			} else if strings.Count(host, ":") == 1 {
				raw = host[:i]
			} else {
				raw = host
			}
		} else {
			raw = host
		}
	}
	// P2.118: rest_is_ip_address gate; invalid REMOTE_ADDR → 127.0.0.1 (WP comments controller).
	if ok := wp.RestIsIPAddress(raw); ok != "" {
		return ok
	}
	return "127.0.0.1"
}

func truncateStr(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}
