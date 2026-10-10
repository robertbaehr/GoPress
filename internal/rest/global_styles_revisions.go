package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// GlobalStylesRevision is GET /wp/v2/global-styles/{parent}/revisions[/id] (P4.108).
type GlobalStylesRevision struct {
	ID          int64          `json:"id"`
	Author      int64          `json:"author"`
	Date        string         `json:"date"`
	DateGMT     string         `json:"date_gmt"`
	Modified    string         `json:"modified"`
	ModifiedGMT string         `json:"modified_gmt"`
	Parent      int64          `json:"parent"`
	Settings    map[string]any `json:"settings"`
	Styles      map[string]any `json:"styles"`
}

func (s *Server) serveGlobalStylesRevisions(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID int64) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	parent, errResp := loadGlobalStylesRevisionParent(c, parentID)
	if errResp != nil {
		writeJSON(w, errResp.status, errResp.body)
		return
	}
	if !canViewGlobalStyles(c) {
		writeGlobalStylesForbidden(w, c, "rest_cannot_read",
			"Sorry, you are not allowed to view revisions for this global style.")
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
	out := make([]GlobalStylesRevision, 0, len(rows))
	if r.Method != http.MethodHead {
		for _, row := range rows {
			rev, ok := globalStylesRevisionFromPost(row, parent.ID)
			if ok {
				out = append(out, rev)
			}
		}
	}
	w.Header().Set("X-WP-Total", strconv.Itoa(len(out)))
	w.Header().Set("X-WP-TotalPages", "1")
	w.Header().Set("Allow", "GET")
	if r.Method == http.MethodHead {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveGlobalStylesRevisionItem(w http.ResponseWriter, r *http.Request, c *wp.Context, parentID, revID int64) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	parent, errResp := loadGlobalStylesRevisionParent(c, parentID)
	if errResp != nil {
		writeJSON(w, errResp.status, errResp.body)
		return
	}
	if !canViewGlobalStyles(c) {
		writeGlobalStylesForbidden(w, c, "rest_cannot_read",
			"Sorry, you are not allowed to view revisions for this global style.")
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
	rev, ok := globalStylesRevisionFromPost(*row, parent.ID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_global_styles_not_found",
			"message": "Cannot find user global styles revisions.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	w.Header().Set("Allow", "GET")
	if r.Method == http.MethodHead {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	writeJSON(w, http.StatusOK, rev)
}

type gsRevErr struct {
	status int
	body   map[string]any
}

func loadGlobalStylesRevisionParent(c *wp.Context, parentID int64) (*wp.Post, *gsRevErr) {
	row, err := wp.GetPost(c, wp.QueryArgs{P: parentID, PostType: "wp_global_styles", PostStatus: "publish"})
	if err != nil {
		return nil, &gsRevErr{status: http.StatusInternalServerError, body: map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		}}
	}
	if row == nil {
		return nil, &gsRevErr{status: http.StatusNotFound, body: map[string]any{
			"code":    "rest_post_invalid_parent",
			"message": "Invalid post parent ID.",
			"data":    map[string]any{"status": 404},
		}}
	}
	if !isGlobalStylesUserThemeJSON(row.Content) {
		return nil, &gsRevErr{status: http.StatusNotFound, body: map[string]any{
			"code":    "rest_global_styles_not_found",
			"message": "Cannot find user global styles revisions.",
			"data":    map[string]any{"status": 404},
		}}
	}
	return row, nil
}

func isGlobalStylesUserThemeJSON(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	var shape struct {
		IsGlobalStylesUserThemeJSON bool `json:"isGlobalStylesUserThemeJSON"`
	}
	if err := json.Unmarshal([]byte(content), &shape); err != nil {
		return false
	}
	return shape.IsGlobalStylesUserThemeJSON
}

func globalStylesRevisionFromPost(row wp.Post, parentID int64) (GlobalStylesRevision, bool) {
	if !isGlobalStylesUserThemeJSON(row.Content) {
		return GlobalStylesRevision{}, false
	}
	settings, styles := parseGlobalStylesContent(row.Content)
	return GlobalStylesRevision{
		ID:          row.ID,
		Author:      row.Author,
		Date:        mysqlDateToREST(row.Date),
		DateGMT:     mysqlDateToREST(row.DateGMT),
		Modified:    mysqlDateToREST(row.Modified),
		ModifiedGMT: mysqlDateToREST(row.ModifiedGMT),
		Parent:      parentID,
		Settings:    settings,
		Styles:      styles,
	}, true
}
