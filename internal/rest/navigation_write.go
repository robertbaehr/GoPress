package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// createNavigation handles POST /wp/v2/navigation (P4.38).
func (s *Server) createNavigation(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create navigation menus as this user.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !canViewMenus(c) {
		writeMenusForbidden(w, c)
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
		status = "publish"
	}
	title := rawOrRendered(body.Title)
	content := rawOrRendered(body.Content)
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		slug = wp.SanitizeTitle(title)
	} else {
		slug = wp.SanitizeTitle(slug)
	}
	if slug == "" {
		slug = "navigation"
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
		Excerpt: "",
		Status:  status,
		Slug:    slug,
		Type:    "wp_navigation",
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_cannot_create",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	row, err := loadPostRow(c, "wp_navigation", id)
	if err != nil || row == nil {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id})
		return
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusCreated, navigationFromWP(*row, home))
}
