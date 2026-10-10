package rest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Autosave is a wp/v2 autosave object (GET collection/item, view context).
// Preview_link is edit-context only in WP, so it is omitted here.
type Autosave struct {
	Author      int64          `json:"author"`
	Date        string         `json:"date"`
	DateGMT     string         `json:"date_gmt"`
	ID          int64          `json:"id"`
	Modified    string         `json:"modified"`
	ModifiedGMT string         `json:"modified_gmt"`
	Parent      int64          `json:"parent"`
	Slug        string         `json:"slug"`
	GUID        RenderedString `json:"guid"`
	Title       RenderedString `json:"title"`
	Content     RenderedString `json:"content"`
	Excerpt     RenderedString `json:"excerpt"`
	Meta        map[string]any `json:"meta"`
	Links       map[string]any `json:"_links"`
}

func (s *Server) servePostAutosaves(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID int64) {
	if r.Method == http.MethodPost {
		s.createPostAutosave(w, r, c, parentID)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	parent, err := wp.GetPostByID(c, parentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if parent == nil || parent.Type != "post" {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_parent",
			"message": "Invalid post parent ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "edit_post", parentID) {
		status := http.StatusUnauthorized
		if c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_read",
			"message": "Sorry, you are not allowed to view autosaves of this post.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	rows, err := listPostAutosaves(c, parentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_autosave_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]Autosave, 0, len(rows))
	for _, row := range rows {
		out = append(out, autosaveFromWP(c, row, home, parentID))
	}
	// WP autosaves collection does not emit X-WP-Total headers.
	w.Header().Set("Allow", "GET, POST")
	writeJSON(w, http.StatusOK, out)
}

// createPostAutosave handles POST /wp/v2/posts/{id}/autosaves (P4.69).
// Draft posts owned by the current user update in place; otherwise a per-user
// revision with post_name "{id}-autosave-v1" is created or overwritten.
func (s *Server) createPostAutosave(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit this post.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	parent, err := loadPostRow(c, "post", parentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if parent == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_parent",
			"message": "Invalid post parent ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_post", parentID) {
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
	title := parent.Title
	content := parent.Content
	excerpt := parent.Excerpt
	if len(body.Title) > 0 {
		title = rawOrRendered(body.Title)
	}
	if len(body.Content) > 0 {
		content = rawOrRendered(body.Content)
	}
	if len(body.Excerpt) > 0 {
		excerpt = rawOrRendered(body.Excerpt)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	userID := c.CurrentUser.ID
	isDraft := parent.Status == "draft" || parent.Status == "auto-draft"
	if isDraft && parent.Author == userID {
		if err := updatePostRow(c, parentID, title, content, excerpt, parent.Name, parent.Status, now); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_autosave_update",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
		row, err := loadPostRow(c, "post", parentID)
		if err != nil || row == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_autosave_query",
				"message": "autosave updated but could not be reloaded",
				"data":    map[string]any{"status": 500},
			})
			return
		}
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusOK, autosaveFromWP(c, *row, home, parentID))
		return
	}

	old, err := findUserAutosave(c, parentID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_autosave_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	different := normalizeWS(title) != normalizeWS(parent.Title) ||
		normalizeWS(content) != normalizeWS(parent.Content) ||
		normalizeWS(excerpt) != normalizeWS(parent.Excerpt)
	if !different && old != nil {
		home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusOK, autosaveFromWP(c, *old, home, parentID))
		return
	}

	var autoID int64
	slug := fmt.Sprintf("%d-autosave-v1", parentID)
	if old != nil {
		if err := updateRevisionRow(c, old.ID, userID, title, content, excerpt, now); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_autosave_update",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
		autoID = old.ID
	} else {
		autoID, err = insertPostRow(c, postRowInsert{
			Author:  userID,
			Date:    now,
			Content: content,
			Title:   title,
			Excerpt: excerpt,
			Status:  "inherit",
			Slug:    slug,
			Type:    "revision",
			Parent:  parentID,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_autosave_create",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
	}
	row, err := wp.GetPostByID(c, autoID)
	if err != nil || row == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_autosave_query",
			"message": "autosave saved but could not be reloaded",
			"data":    map[string]any{"status": 500},
		})
		return
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	w.Header().Set("Allow", "GET, POST")
	writeJSON(w, http.StatusOK, autosaveFromWP(c, *row, home, parentID))
}

func findUserAutosave(c *wp.Context, parentID, userID int64) (*wp.Post, error) {
	if c == nil || c.DB == nil || parentID <= 0 || userID <= 0 {
		return nil, nil
	}
	slug := fmt.Sprintf("%d-autosave-v1", parentID)
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, "+
			"post_name, post_status, post_type, post_title, post_content, post_excerpt, "+
			"guid, comment_status, ping_status, post_password, post_parent, menu_order FROM "+
			c.DB.Posts()+
			" WHERE post_type = %s AND post_parent = %d AND post_name = %s AND post_author = %d"+
			" ORDER BY post_modified DESC, ID DESC LIMIT 1",
		"revision", parentID, slug, userID,
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
		ID:          asInt64(row["ID"]),
		Author:      asInt64(row["post_author"]),
		Date:        asString(row["post_date"]),
		DateGMT:     asString(row["post_date_gmt"]),
		Modified:    asString(row["post_modified"]),
		ModifiedGMT: asString(row["post_modified_gmt"]),
		Name:        asString(row["post_name"]),
		Status:      asString(row["post_status"]),
		Type:        asString(row["post_type"]),
		Title:       asString(row["post_title"]),
		Content:     asString(row["post_content"]),
		Excerpt:     asString(row["post_excerpt"]),
		GUID:        asString(row["guid"]),
		Parent:      asInt64(row["post_parent"]),
	}
	return &p, nil
}

func updateRevisionRow(c *wp.Context, id, author int64, title, content, excerpt, modified string) error {
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_author = %d, post_title = %s, post_content = %s, post_excerpt = %s, "+
			"post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		author, title, content, excerpt, modified, modified, id, "revision",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func normalizeWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func (s *Server) servePostAutosaveItem(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID, autoID int64) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	parent, err := wp.GetPostByID(c, parentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if parent == nil || parent.Type != "post" {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_parent",
			"message": "Invalid post parent ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "edit_post", parentID) {
		status := http.StatusUnauthorized
		if c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_read",
			"message": "Sorry, you are not allowed to view autosaves of this post.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	row, err := wp.GetPostByID(c, autoID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_autosave_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if row == nil || row.Type != "revision" || row.Parent != parentID || !strings.Contains(row.Name, "autosave") {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid autosave ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, autosaveFromWP(c, *row, home, parentID))
}

func listPostAutosaves(c *wp.Context, parentID int64) ([]wp.Post, error) {
	if c == nil || c.DB == nil || parentID <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, "+
			"post_name, post_status, post_type, post_title, post_content, post_excerpt, "+
			"guid, comment_status, ping_status, post_password, post_parent, menu_order FROM "+
			c.DB.Posts()+
			" WHERE post_type = %s AND post_parent = %d AND post_name LIKE %s"+
			" ORDER BY post_modified DESC, ID DESC",
		"revision", parentID, "%autosave%",
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	out := make([]wp.Post, 0, len(rows))
	for _, row := range rows {
		out = append(out, wp.Post{
			ID:          asInt64(row["ID"]),
			Author:      asInt64(row["post_author"]),
			Date:        asString(row["post_date"]),
			DateGMT:     asString(row["post_date_gmt"]),
			Modified:    asString(row["post_modified"]),
			ModifiedGMT: asString(row["post_modified_gmt"]),
			Name:        asString(row["post_name"]),
			Status:      asString(row["post_status"]),
			Type:        asString(row["post_type"]),
			Title:       asString(row["post_title"]),
			Content:     asString(row["post_content"]),
			Excerpt:     asString(row["post_excerpt"]),
			GUID:        asString(row["guid"]),
			Parent:      asInt64(row["post_parent"]),
		})
	}
	return out, nil
}

func autosaveFromWP(c *wp.Context, row wp.Post, home string, parentID int64) Autosave {
	rendered := renderPostContentCtx(c, row.Content)
	titleRendered := restTexturize(row.Title)
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	return Autosave{
		Author:      row.Author,
		Date:        mysqlDateToREST(row.Date),
		DateGMT:     mysqlDateToREST(row.DateGMT),
		ID:          row.ID,
		Modified:    mysqlDateToREST(row.Modified),
		ModifiedGMT: mysqlDateToREST(row.ModifiedGMT),
		Parent:      parentID,
		Slug:        row.Name,
		GUID:        RenderedString{Rendered: row.GUID},
		Title:       RenderedString{Rendered: titleRendered},
		Content:     RenderedString{Rendered: rendered},
		Excerpt:     RenderedString{Rendered: revisionExcerptRendered(row.Excerpt)},
		Meta:        map[string]any{"footnotes": ""},
		Links: map[string]any{
			"parent": []map[string]any{{
				"href": fmt.Sprintf("%s/posts/%d", restRoot, parentID),
			}},
		},
	}
}
