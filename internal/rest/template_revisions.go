package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// TemplateRevision is GET /wp/v2/templates/{parent}/revisions[/id] (P4.111).
// Shape matches a BlockTemplate plus parent (revision post parent ID).
type TemplateRevision struct {
	ID           string            `json:"id"`
	Theme        string            `json:"theme"`
	Slug         string            `json:"slug"`
	Source       string            `json:"source"`
	Type         string            `json:"type"`
	Title        map[string]string `json:"title"`
	Content      map[string]any    `json:"content"`
	Status       string            `json:"status"`
	WPID         int64             `json:"wp_id"`
	HasThemeFile bool              `json:"has_theme_file"`
	IsCustom     bool              `json:"is_custom,omitempty"`
	Parent       int64             `json:"parent"`
}

// parseTemplateRevisionsPath matches {id}/revisions[/revID] under templates or template-parts.
func parseTemplateRevisionsPath(path, prefix string) (parentID string, revID int64, hasRev bool, ok bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	idx := strings.Index(rest, "/revisions")
	if idx < 0 {
		return "", 0, false, false
	}
	parentID = rest[:idx]
	if parentID == "" {
		return "", 0, false, false
	}
	tail := strings.Trim(rest[idx+len("/revisions"):], "/")
	if tail == "" {
		return parentID, 0, false, true
	}
	if strings.Contains(tail, "/") {
		return "", 0, false, false
	}
	id, err := strconv.ParseInt(tail, 10, 64)
	if err != nil || id <= 0 {
		return "", 0, false, false
	}
	return parentID, id, true, true
}

func (s *Server) serveTemplateRevisions(w http.ResponseWriter, r *http.Request, c *wp.Context, parentTemplateID string) {
	s.serveBlockTemplateRevisions(w, r, c, "wp_template", parentTemplateID)
}

func (s *Server) serveTemplatePartRevisions(w http.ResponseWriter, r *http.Request, c *wp.Context, parentTemplateID string) {
	s.serveBlockTemplateRevisions(w, r, c, "wp_template_part", parentTemplateID)
}

func (s *Server) serveTemplateRevisionItem(w http.ResponseWriter, r *http.Request, c *wp.Context, parentTemplateID string, revID int64) {
	s.serveBlockTemplateRevisionItem(w, r, c, "wp_template", parentTemplateID, revID)
}

func (s *Server) serveTemplatePartRevisionItem(w http.ResponseWriter, r *http.Request, c *wp.Context, parentTemplateID string, revID int64) {
	s.serveBlockTemplateRevisionItem(w, r, c, "wp_template_part", parentTemplateID, revID)
}

func (s *Server) serveBlockTemplateRevisions(w http.ResponseWriter, r *http.Request, c *wp.Context, postType, parentTemplateID string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	parent, errResp := loadTemplateRevisionParent(c, postType, parentTemplateID)
	if errResp != nil {
		writeJSON(w, errResp.status, errResp.body)
		return
	}
	if !canViewTemplates(c) {
		writeTemplatesForbidden(w, c)
		return
	}
	rows, err := listPostRevisions(c, parent.WPID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_revision_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	out := make([]TemplateRevision, 0, len(rows))
	if r.Method != http.MethodHead {
		for _, row := range rows {
			out = append(out, templateRevisionFromPost(c, row, parent))
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

func (s *Server) serveBlockTemplateRevisionItem(w http.ResponseWriter, r *http.Request, c *wp.Context, postType, parentTemplateID string, revID int64) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	parent, errResp := loadTemplateRevisionParent(c, postType, parentTemplateID)
	if errResp != nil {
		writeJSON(w, errResp.status, errResp.body)
		return
	}
	if !canViewTemplates(c) {
		writeTemplatesForbidden(w, c)
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
	if row == nil || row.Type != "revision" || row.Parent != parent.WPID {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid revision ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	w.Header().Set("Allow", "GET")
	if r.Method == http.MethodHead {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	writeJSON(w, http.StatusOK, templateRevisionFromPost(c, *row, parent))
}

type tplRevErr struct {
	status int
	body   map[string]any
}

func loadTemplateRevisionParent(c *wp.Context, postType, parentTemplateID string) (*BlockTemplate, *tplRevErr) {
	parentTemplateID = strings.Trim(parentTemplateID, "/")
	var found *BlockTemplate
	for _, t := range listBlockTemplates(c, postType) {
		if t.ID == parentTemplateID || t.Slug == parentTemplateID {
			cp := t
			found = &cp
			break
		}
	}
	if found == nil {
		return nil, &tplRevErr{status: http.StatusNotFound, body: map[string]any{
			"code":    "rest_post_invalid_parent",
			"message": "Invalid template parent ID.",
			"data":    map[string]any{"status": 404},
		}}
	}
	if found.WPID <= 0 {
		return nil, &tplRevErr{status: http.StatusBadRequest, body: map[string]any{
			"code":    "rest_invalid_template",
			"message": "Templates based on theme files can't have revisions.",
			"data":    map[string]any{"status": 400},
		}}
	}
	return found, nil
}

func templateRevisionFromPost(c *wp.Context, row wp.Post, parent *BlockTemplate) TemplateRevision {
	stylesheet := wp.ActiveStylesheet(c)
	title := row.Title
	if title == "" {
		title = humanizeSlug(row.Name)
	}
	slug := parent.Slug
	if slug == "" {
		slug = row.Name
	}
	return TemplateRevision{
		ID:           stylesheet + "//" + slug,
		Theme:        stylesheet,
		Slug:         slug,
		Source:       "custom",
		Type:         parent.Type,
		Title:        map[string]string{"raw": title, "rendered": title},
		Content:      map[string]any{"raw": row.Content, "block_version": 1},
		Status:       "publish",
		WPID:         row.ID,
		HasThemeFile: parent.HasThemeFile,
		IsCustom:     parent.IsCustom,
		Parent:       parent.WPID,
	}
}
