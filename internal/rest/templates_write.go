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

// templateWriteBody is the editable subset of WP_REST_Templates_Controller (P4.52).
type templateWriteBody struct {
	Slug    string          `json:"slug"`
	Theme   string          `json:"theme"`
	Title   json.RawMessage `json:"title"`
	Content json.RawMessage `json:"content"`
	Status  string          `json:"status"`
	Source  string          `json:"source"`
	Area    string          `json:"area"`
}

func canEditTemplates(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, "edit_theme_options")
}

func writeTemplatesCannotEdit(w http.ResponseWriter, c *wp.Context) {
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]any{
		"code":    "rest_cannot_manage_templates",
		"message": "Sorry, you are not allowed to access the templates on this site.",
		"data":    map[string]any{"status": status},
	})
}

func (s *Server) createBlockTemplate(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string) {
	if !canEditTemplates(c) {
		writeTemplatesCannotEdit(w, c)
		return
	}
	body, err := decodeTemplateWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_missing_callback_param",
			"message": "Missing parameter(s): slug",
			"data":    map[string]any{"status": 400, "params": []string{"slug"}},
		})
		return
	}
	slug = wp.SanitizeTitle(slug)
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): slug",
			"data":    map[string]any{"status": 400},
		})
		return
	}
	title := rawOrRendered(body.Title)
	if title == "" {
		title = humanizeSlug(slug)
	}
	content := rawOrRendered(body.Content)
	status := body.Status
	if status == "" {
		status = "publish"
	}
	author := int64(0)
	if c.CurrentUser != nil {
		author = c.CurrentUser.ID
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	id, err := insertPostRow(c, postRowInsert{
		Author:  author,
		Date:    now,
		Content: content,
		Title:   title,
		Excerpt: "",
		Status:  status,
		Slug:    slug,
		Type:    postType,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_template_insert_error",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	stylesheet := wp.ActiveStylesheet(c)
	if body.Theme != "" {
		stylesheet = body.Theme
	}
	bt := findBlockTemplateBySlug(c, postType, slug)
	if bt == nil {
		area := ""
		if postType == "wp_template_part" {
			area = body.Area
			if area == "" {
				area = inferTemplatePartArea(slug)
			}
		}
		bt = &BlockTemplate{
			ID:           stylesheet + "//" + slug,
			Theme:        stylesheet,
			Slug:         slug,
			Source:       "custom",
			Type:         postType,
			Title:        map[string]string{"raw": title, "rendered": title},
			Content:      map[string]any{"raw": content, "block_version": 1},
			Status:       status,
			WPID:         id,
			HasThemeFile: false,
			IsCustom:     postType == "wp_template",
			Area:         area,
		}
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, bt)
}

func (s *Server) updateBlockTemplate(w http.ResponseWriter, r *http.Request, c *wp.Context, postType, id string) {
	if !canEditTemplates(c) {
		writeTemplatesCannotEdit(w, c)
		return
	}
	id = strings.Trim(id, "/")
	existing := findBlockTemplate(c, postType, id)
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_template_not_found",
			"message": "No templates exist with that id.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	body, err := decodeTemplateWriteBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}

	title := existing.Title["raw"]
	if body.Title != nil {
		if t := rawOrRendered(body.Title); t != "" {
			title = t
		}
	}
	content := ""
	if raw, ok := existing.Content["raw"].(string); ok {
		content = raw
	}
	if body.Content != nil {
		content = rawOrRendered(body.Content)
	}
	status := existing.Status
	if body.Status != "" {
		status = body.Status
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	if existing.Source == "custom" && existing.WPID > 0 {
		if err := updateTypedPostRow(c, existing.WPID, postType, title, content, existing.Slug, status, now); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_template_update_error",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
	} else {
		// Theme-sourced template: insert a custom override (WP parity).
		author := int64(0)
		if c.CurrentUser != nil {
			author = c.CurrentUser.ID
		}
		wpID, err := insertPostRow(c, postRowInsert{
			Author:  author,
			Date:    now,
			Content: content,
			Title:   title,
			Excerpt: "",
			Status:  status,
			Slug:    existing.Slug,
			Type:    postType,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"code":    "rest_template_insert_error",
				"message": err.Error(),
				"data":    map[string]any{"status": 500},
			})
			return
		}
		existing.WPID = wpID
		existing.Source = "custom"
		existing.IsCustom = postType == "wp_template" && !existing.HasThemeFile
	}

	updated := findBlockTemplate(c, postType, id)
	if updated == nil {
		existing.Title = map[string]string{"raw": title, "rendered": title}
		existing.Content = map[string]any{"raw": content, "block_version": 1}
		existing.Status = status
		updated = existing
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, updated)
}

func findBlockTemplate(c *wp.Context, postType, id string) *BlockTemplate {
	for _, t := range listBlockTemplates(c, postType) {
		if t.ID == id || t.Slug == id {
			cp := t
			return &cp
		}
	}
	return nil
}

func findBlockTemplateBySlug(c *wp.Context, postType, slug string) *BlockTemplate {
	for _, t := range listBlockTemplates(c, postType) {
		if t.Slug == slug {
			cp := t
			return &cp
		}
	}
	return nil
}

func decodeTemplateWriteBody(r *http.Request) (templateWriteBody, error) {
	var body templateWriteBody
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

func updateTypedPostRow(c *wp.Context, id int64, postType, title, content, slug, status, modified string) error {
	if c == nil || c.DB == nil {
		return fmt.Errorf("nil db")
	}
	q, args, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+
			" SET post_title = %s, post_content = %s, post_name = %s, post_status = %s, "+
			"post_modified = %s, post_modified_gmt = %s WHERE ID = %d AND post_type = %s",
		title, content, slug, status, modified, modified, id, postType,
	)
	if err != nil {
		return err
	}
	_, err = c.DB.Exec(q, args...)
	return err
}
