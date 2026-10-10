package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Revision is a wp/v2 revision object (GET collection/item).
// View-context revisions omit content/excerpt.protected (unlike posts).
type Revision struct {
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

func (s *Server) servePostRevisions(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID int64) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	// P4.115: OPTIONS returns public item schema (like WP_REST_Revisions_Controller).
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, revisionItemSchema())
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
			"message": "Sorry, you are not allowed to view revisions of this post.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	if !validateRevisionContext(w, r) {
		return
	}
	rows, err := listPostRevisions(c, parentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_revision_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	total := len(rows)
	page, perPage, offset, ok := parseRevisionPagination(w, r, total)
	if !ok {
		return
	}
	_ = page
	sliced := rows
	if perPage > 0 {
		start := offset
		if start > total {
			start = total
		}
		end := start + perPage
		if end > total {
			end = total
		}
		sliced = rows[start:end]
	}
	totalPages := 1
	if perPage > 0 && total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]Revision, 0, len(sliced))
	for _, row := range sliced {
		out = append(out, revisionFromWP(c, row, home, parentID))
	}
	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	w.Header().Set("Allow", "GET")
	// P4.115: HEAD returns totals with empty body.
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) servePostRevisionItem(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID, revID int64) {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		s.getPostRevisionItem(w, r, c, parentID, revID)
	case http.MethodDelete:
		s.deletePostRevision(w, r, c, parentID, revID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func (s *Server) getPostRevisionItem(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID, revID int64) {
	// P4.115: OPTIONS on item returns the same public schema.
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET, DELETE")
		writeJSON(w, http.StatusOK, revisionItemSchema())
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
			"message": "Sorry, you are not allowed to view revisions of this post.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	if !validateRevisionContext(w, r) {
		return
	}
	row, err := wp.GetPostByID(c, revID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_revision_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	// WP revisions endpoint returns both regular revisions and autosaves.
	if row == nil || row.Type != "revision" || row.Parent != parentID {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid revision ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	w.Header().Set("Allow", "GET, DELETE")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, revisionFromWP(c, *row, home, parentID))
}

// deletePostRevision handles DELETE /wp/v2/posts/{parent}/revisions/{id} (P4.68).
// Revisions do not support trashing; force=true is required.
func (s *Server) deletePostRevision(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID, revID int64) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete revisions of this post.",
			"data":    map[string]any{"status": 401},
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
	if !c.UserCan(c.CurrentUser, "delete_post", parentID) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "Sorry, you are not allowed to delete revisions of this post.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	row, err := wp.GetPostByID(c, revID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_revision_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if row == nil || row.Type != "revision" || row.Parent != parentID {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid revision ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	// WP map_meta_cap returns do_not_allow for delete_post on revision IDs;
	// authorize via parent (checked above), matching WP REST's effective gate.
	force := r.URL.Query().Get("force") == "true"
	if !force {
		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"code":    "rest_trash_not_supported",
			"message": "Revisions do not support trashing. Set 'force=true' to delete.",
			"data":    map[string]any{"status": 501},
		})
		return
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	previous := revisionFromWP(c, *row, home, parentID)
	if err := hardDeleteRevision(c, revID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_delete",
			"message": "The post cannot be deleted.",
			"data":    map[string]any{"status": 500},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":  true,
		"previous": previous,
	})
}

func hardDeleteRevision(c *wp.Context, id int64) error {
	if c == nil || c.DB == nil || id <= 0 {
		return fmt.Errorf("invalid delete")
	}
	q, args, err := c.DB.Prepare(
		"DELETE FROM "+c.DB.Posts()+" WHERE ID = %d AND post_type = %s",
		id, "revision",
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}

func listPostRevisions(c *wp.Context, parentID int64) ([]wp.Post, error) {
	if c == nil || c.DB == nil || parentID <= 0 {
		return nil, nil
	}
	// Match WP_Query post_type=revision post_parent=N — includes autosaves.
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, "+
			"post_name, post_status, post_type, post_title, post_content, post_excerpt, "+
			"guid, comment_status, ping_status, post_password, post_parent, menu_order FROM "+
			c.DB.Posts()+
			" WHERE post_type = %s AND post_parent = %d"+
			" ORDER BY post_date DESC, ID DESC",
		"revision", parentID,
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

func countPostRevisions(c *wp.Context, parentID int64) int {
	rows, err := listPostRevisions(c, parentID)
	if err != nil {
		return 0
	}
	return len(rows)
}

func revisionFromWP(c *wp.Context, row wp.Post, home string, parentID int64) Revision {
	rendered := renderPostContentCtx(c, row.Content)
	titleRendered := row.Title
	if titleRendered != "" {
		titleRendered = restTexturize(titleRendered)
	}
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	return Revision{
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

// revisionExcerptRendered mirrors WP_REST_Revisions_Controller::prepare_excerpt_response:
// empty post_excerpt stays empty (no wp_trim_excerpt fallback).
func revisionExcerptRendered(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	ex := restTexturize(raw)
	if !strings.HasPrefix(ex, "<") {
		ex = "<p>" + ex + "</p>\n"
	} else if !strings.HasSuffix(ex, "\n") {
		ex += "\n"
	}
	return ex
}

func revisionContext(r *http.Request) string {
	if r == nil {
		return "view"
	}
	ctx := r.URL.Query().Get("context")
	if ctx == "" {
		return "view"
	}
	return ctx
}

func validateRevisionContext(w http.ResponseWriter, r *http.Request) bool {
	switch revisionContext(r) {
	case "view", "edit", "embed":
		return true
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): context",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{"context": "context is not one of view, edit, embed."},
			},
		})
		return false
	}
}

// parseRevisionPagination mirrors WP revisions collection page/per_page/offset (P4.115).
// When per_page is omitted, all revisions are returned (WP unsets the default).
func parseRevisionPagination(w http.ResponseWriter, r *http.Request, total int) (page, perPage, offset int, ok bool) {
	page = 1
	perPage = 0 // 0 = all
	offset = 0
	q := r.URL.Query()
	if v := strings.TrimSpace(q.Get("per_page")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): per_page",
				"data":    map[string]any{"status": 400},
			})
			return 0, 0, 0, false
		}
		if n > 100 {
			n = 100
		}
		perPage = n
	}
	if v := strings.TrimSpace(q.Get("page")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): page",
				"data":    map[string]any{"status": 400},
			})
			return 0, 0, 0, false
		}
		page = n
	}
	if v := strings.TrimSpace(q.Get("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): offset",
				"data":    map[string]any{"status": 400},
			})
			return 0, 0, 0, false
		}
		offset = n
	} else if perPage > 0 {
		offset = (page - 1) * perPage
	}
	if perPage > 0 && total > 0 {
		maxPages := (total + perPage - 1) / perPage
		if offset > 0 && offset >= total {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_revision_invalid_offset_number",
				"message": "The offset number requested is larger than or equal to the number of available revisions.",
				"data":    map[string]any{"status": 400},
			})
			return 0, 0, 0, false
		}
		if offset == 0 && page > maxPages {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_revision_invalid_page_number",
				"message": "The page number requested is larger than the number of pages available.",
				"data":    map[string]any{"status": 400},
			})
			return 0, 0, 0, false
		}
	}
	return page, perPage, offset, true
}

func revisionItemSchema() map[string]any {
	ctxAll := []string{"view", "edit", "embed"}
	return map[string]any{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"title":   "post-revision",
		"type":    "object",
		"properties": map[string]any{
			"author":       map[string]any{"description": "The ID for the author of the revision.", "type": "integer", "context": ctxAll},
			"date":         map[string]any{"description": "The date the revision was published, in the site's timezone.", "type": "string", "format": "date-time", "context": ctxAll},
			"date_gmt":     map[string]any{"description": "The date the revision was published, as GMT.", "type": "string", "format": "date-time", "context": ctxAll},
			"id":           map[string]any{"description": "Unique identifier for the revision.", "type": "integer", "context": ctxAll},
			"modified":     map[string]any{"description": "The date the revision was last modified, in the site's timezone.", "type": "string", "format": "date-time", "context": ctxAll},
			"modified_gmt": map[string]any{"description": "The date the revision was last modified, as GMT.", "type": "string", "format": "date-time", "context": ctxAll},
			"parent":       map[string]any{"description": "The ID for the parent of the revision.", "type": "integer", "context": ctxAll},
			"slug":         map[string]any{"description": "An alphanumeric identifier for the revision unique to its type.", "type": "string", "context": ctxAll},
			"guid":         map[string]any{"description": "The globally unique identifier for the post.", "type": "object", "context": ctxAll},
			"title":        map[string]any{"description": "The title for the post.", "type": "object", "context": ctxAll},
			"content":      map[string]any{"description": "The content for the post.", "type": "object", "context": ctxAll},
			"excerpt":      map[string]any{"description": "The excerpt for the post.", "type": "object", "context": ctxAll},
			"meta":         map[string]any{"description": "Meta fields.", "type": "object", "context": ctxAll},
		},
	}
}
