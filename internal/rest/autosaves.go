package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Autosave is a wp/v2 autosave object (GET stub).
type Autosave struct {
	ID          int64             `json:"id"`
	Author      int64             `json:"author"`
	Date        string            `json:"date"`
	DateGMT     string            `json:"date_gmt"`
	Modified    string            `json:"modified"`
	ModifiedGMT string            `json:"modified_gmt"`
	Parent      int64             `json:"parent"`
	Slug        string            `json:"slug"`
	GUID        RenderedString    `json:"guid"`
	Title       RenderedString    `json:"title"`
	Content     ProtectedRendered `json:"content"`
	Excerpt     ProtectedRendered `json:"excerpt"`
	PreviewLink string            `json:"preview_link"`
	Links       map[string]any    `json:"_links"`
}

func (s *Server) servePostAutosaves(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID int64) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		// POST create is not implemented in this stub.
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
		out = append(out, autosaveFromWP(row, home, parentID))
	}
	w.Header().Set("X-WP-Total", strconv.Itoa(len(out)))
	w.Header().Set("X-WP-TotalPages", "1")
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, out)
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
	writeJSON(w, http.StatusOK, autosaveFromWP(*row, home, parentID))
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

func autosaveFromWP(row wp.Post, home string, parentID int64) Autosave {
	rendered := renderPostContent(row.Content) // stub: no request context for shortcodes
	titleRendered := restTexturize(row.Title)
	excerpt := buildExcerpt(row.Excerpt, rendered, false)
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	preview := strings.TrimRight(home, "/") + "/?p=" + strconv.FormatInt(parentID, 10) + "&preview=true"
	return Autosave{
		ID:          row.ID,
		Author:      row.Author,
		Date:        mysqlDateToREST(row.Date),
		DateGMT:     mysqlDateToREST(row.DateGMT),
		Modified:    mysqlDateToREST(row.Modified),
		ModifiedGMT: mysqlDateToREST(row.ModifiedGMT),
		Parent:      parentID,
		Slug:        row.Name,
		GUID:        RenderedString{Rendered: row.GUID},
		Title:       RenderedString{Rendered: titleRendered},
		Content:     ProtectedRendered{Rendered: rendered, Protected: false},
		Excerpt:     excerpt,
		PreviewLink: preview,
		Links: map[string]any{
			"self": []map[string]any{{
				"href": fmt.Sprintf("%s/posts/%d/autosaves/%d", restRoot, parentID, row.ID),
			}},
			"parent": []map[string]any{{
				"href": fmt.Sprintf("%s/posts/%d", restRoot, parentID),
			}},
		},
	}
}
