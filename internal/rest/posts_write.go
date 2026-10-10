package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

type postWriteBody struct {
	Title   json.RawMessage `json:"title"`
	Content json.RawMessage `json:"content"`
	Excerpt json.RawMessage `json:"excerpt"`
	Status  string          `json:"status"`
	Slug    string          `json:"slug"`
	Author  *int64          `json:"author"`
}

func (s *Server) servePostCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string) {
	switch r.Method {
	case http.MethodGet:
		s.listPostCollection(w, r, c, postType)
	case http.MethodPost:
		if postType != "post" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 405},
			})
			return
		}
		s.createPost(w, r, c)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) listPostCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string) {
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	offset, useOffset := parsePostsOffset(r)
	edit := r.URL.Query().Get("context") == "edit" && canEditPosts(c)
	stickyFilter := parseStickyQuery(r)
	orderby := parsePostsOrderby(r)
	order := parsePostsOrder(r)
	search := parsePostsSearch(r)
	author := parsePostsAuthor(r)
	categories := parsePostsCategories(r)
	tags := parsePostsTags(r)
	slugs := parsePostsSlug(r)
	include := parsePostsInclude(r)
	exclude := parsePostsExclude(r)
	dateAfter := parsePostsAfter(r)
	dateBefore := parsePostsBefore(r)
	modifiedAfter := parsePostsModifiedAfter(r)
	modifiedBefore := parsePostsModifiedBefore(r)
	statuses, statusCode, statusMsg := parsePostsStatus(r, c)
	if statusCode != 0 {
		writeJSON(w, statusCode, map[string]any{
			"code":    "rest_forbidden_status",
			"message": statusMsg,
			"data":    map[string]any{"status": statusCode},
		})
		return
	}
	total, err := countPosts(c, postType, stickyFilter, search, author, categories, tags, statuses, slugs, include, exclude, dateAfter, dateBefore, modifiedAfter, modifiedBefore)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	posts, err := listPostsPaged(c, postType, perPage, page, edit, stickyFilter, orderby, order, search, author, categories, tags, statuses, slugs, include, exclude, dateAfter, dateBefore, modifiedAfter, modifiedBefore, offset, useOffset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
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
	if canEditPosts(c) {
		w.Header().Set("Allow", "GET, POST")
	}
	writeJSON(w, http.StatusOK, posts)
}

func (s *Server) serveTypedItem(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string, id int64) {
	switch r.Method {
	case http.MethodGet:
		s.getTypedItem(w, r, c, postType, id)
	case http.MethodPut, http.MethodPost, http.MethodPatch:
		if postType != "post" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 405},
			})
			return
		}
		s.updatePost(w, r, c, id)
	case http.MethodDelete:
		if postType != "post" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 405},
			})
			return
		}
		s.deletePost(w, r, c, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) getTypedItem(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string, id int64) {
	edit := r.URL.Query().Get("context") == "edit" && canEditPosts(c)
	var (
		post *Post
		err  error
	)
	// Editors may read drafts/private via any-status lookup (P10.11 e2e create→get).
	if edit {
		post, err = getPostByIDAny(c, postType, id, true)
	} else {
		post, err = getPostByID(c, postType, id, false)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if canEditPosts(c) {
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	}
	writeJSON(w, http.StatusOK, post)
}

func (s *Server) createPost(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create posts as this user.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "create_posts") && !c.UserCan(c.CurrentUser, "edit_posts") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create posts as this user.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	body, err := decodePostWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	status := body.Status
	if status == "" {
		status = "draft"
	}
	if status == "publish" && !c.UserCan(c.CurrentUser, "publish_posts") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_publish",
			"message": "Sorry, you are not allowed to publish posts.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	title := rawOrRendered(body.Title)
	content := rawOrRendered(body.Content)
	excerpt := rawOrRendered(body.Excerpt)
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		slug = wp.SanitizeTitle(title)
	} else {
		slug = wp.SanitizeTitle(slug)
	}
	if slug == "" {
		slug = "post"
	}
	author := c.CurrentUser.ID
	if body.Author != nil && *body.Author > 0 {
		author = *body.Author
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	id, err := insertPostRow(c, postRowInsert{
		Author:  author,
		Date:    now,
		Content: content,
		Title:   title,
		Excerpt: excerpt,
		Status:  status,
		Slug:    slug,
		Type:    "post",
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_create",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	post, err := getPostByIDAny(c, "post", id, true)
	if err != nil || post == nil {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, post)
}

func (s *Server) updatePost(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this post.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	existing, err := loadPostRow(c, "post", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_post", id) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	body, err := decodePostWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	title := existing.Title
	content := existing.Content
	excerpt := existing.Excerpt
	slug := existing.Name
	status := existing.Status
	if body.Title != nil {
		title = rawOrRendered(body.Title)
	}
	if body.Content != nil {
		content = rawOrRendered(body.Content)
	}
	if body.Excerpt != nil {
		excerpt = rawOrRendered(body.Excerpt)
	}
	if body.Slug != "" {
		slug = wp.SanitizeTitle(body.Slug)
	}
	if body.Status != "" {
		if body.Status == "publish" && existing.Status != "publish" && !c.UserCan(c.CurrentUser, "publish_posts") {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"code":    "rest_cannot_publish",
				"message": "Sorry, you are not allowed to publish posts.",
				"data":    map[string]any{"status": 403},
			})
			return
		}
		status = body.Status
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	changed := title != existing.Title || content != existing.Content || excerpt != existing.Excerpt
	if err := updatePostRow(c, id, title, content, excerpt, slug, status, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_update",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	// Create a revision of the updated post when revisioned fields change (P4.48).
	if changed && status != "auto-draft" {
		_, _ = savePostRevision(c, existing.Author, id, title, content, excerpt, now)
	}
	post, err := getPostByIDAny(c, "post", id, true)
	if err != nil || post == nil {
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, post)
}

func (s *Server) deletePost(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete this post.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	existing, err := loadPostRow(c, "post", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "delete_post", id) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete this post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if err := updatePostStatus(c, id, "trash", now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_delete",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	post, err := getPostByIDAny(c, "post", id, true)
	if err != nil || post == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"deleted":  true,
			"previous": map[string]any{"id": id, "status": existing.Status},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":  true,
		"previous": post,
	})
}

func decodePostWriteBody(r *http.Request) (postWriteBody, error) {
	var body postWriteBody
	if r == nil || r.Body == nil {
		return body, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return body, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return body, nil
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return body, fmt.Errorf("invalid JSON body")
	}
	return body, nil
}

func rawOrRendered(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		Raw      string `json:"raw"`
		Rendered string `json:"rendered"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		if obj.Raw != "" {
			return obj.Raw
		}
		return obj.Rendered
	}
	return strings.Trim(string(raw), `"`)
}

type postRowInsert struct {
	Author  int64
	Date    string
	Content string
	Title   string
	Excerpt string
	Status  string
	Slug    string
	Type    string
	Parent  int64
}

// savePostRevision inserts a post_type=revision row for parentID (wp_save_post_revision subset, P4.48).
func savePostRevision(c *wp.Context, author, parentID int64, title, content, excerpt, when string) (int64, error) {
	if parentID <= 0 {
		return 0, fmt.Errorf("invalid parent")
	}
	return insertPostRow(c, postRowInsert{
		Author:  author,
		Date:    when,
		Content: content,
		Title:   title,
		Excerpt: excerpt,
		Status:  "inherit",
		Slug:    fmt.Sprintf("%d-revision-v1", parentID),
		Type:    "revision",
		Parent:  parentID,
	})
}

func insertPostRow(c *wp.Context, in postRowInsert) (int64, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	guid := fmt.Sprintf("%s/?p=0", home)
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		in.Author, in.Date, in.Date, in.Content, in.Title, in.Excerpt, in.Status,
		"open", "open", "", in.Slug, "", "", in.Date, in.Date,
		"", in.Parent, guid, 0, in.Type, "", 0,
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
	// Fix GUID to real ID.
	uq, uargs, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET guid = %s WHERE ID = %d",
		fmt.Sprintf("%s/?p=%d", home, id), id,
	)
	if err == nil {
		_, _ = c.DB.Exec(uq, uargs...)
	}
	return id, nil
}

func updatePostRow(c *wp.Context, id int64, title, content, excerpt, slug, status, modified string) error {
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_title = %s, post_content = %s, post_excerpt = %s, post_name = %s, post_status = %s, "+
			"post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		title, content, excerpt, slug, status, modified, modified, id, "post",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func updatePostStatus(c *wp.Context, id int64, status, modified string) error {
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_status = %s, post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		status, modified, modified, id, "post",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func loadPostRow(c *wp.Context, postType string, id int64) (*wp.Post, error) {
	if c == nil || c.DB == nil || id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, "+
			"post_name, post_status, post_type, post_title, post_content, post_excerpt, "+
			"guid, comment_status, ping_status, post_password, post_parent, menu_order FROM "+c.DB.Posts()+
			" WHERE ID = %d AND post_type = %s LIMIT 1",
		id, postType,
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
	p := wp.Post{
		ID:            asInt64(row["ID"]),
		Author:        asInt64(row["post_author"]),
		Date:          asString(row["post_date"]),
		DateGMT:       asString(row["post_date_gmt"]),
		Modified:      asString(row["post_modified"]),
		ModifiedGMT:   asString(row["post_modified_gmt"]),
		Name:          asString(row["post_name"]),
		Status:        asString(row["post_status"]),
		Type:          asString(row["post_type"]),
		Title:         asString(row["post_title"]),
		Content:       asString(row["post_content"]),
		Excerpt:       asString(row["post_excerpt"]),
		GUID:          asString(row["guid"]),
		CommentStatus: asString(row["comment_status"]),
		PingStatus:    asString(row["ping_status"]),
		Password:      asString(row["post_password"]),
		Parent:        asInt64(row["post_parent"]),
		MenuOrder:     int(asInt64(row["menu_order"])),
	}
	return &p, nil
}

func getPostByIDAny(c *wp.Context, postType string, id int64, edit bool) (*Post, error) {
	row, err := loadPostRow(c, postType, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	p := postFromWP(c, *row, home, edit, stickyPostIDs(c))
	return &p, nil
}
