package rest

import (
	"net/http"
	"sort"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// RESTBlockPattern is a wp/v2/block-patterns/patterns item (P5.38 / P4.106).
// Field names follow the WP REST schema (snake_case).
type RESTBlockPattern struct {
	Name          string   `json:"name"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Description   string   `json:"description,omitempty"`
	Inserter      bool     `json:"inserter"`
	Categories    []string `json:"categories,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	BlockTypes    []string `json:"block_types,omitempty"`
	PostTypes     []string `json:"post_types,omitempty"`
	TemplateTypes []string `json:"template_types,omitempty"`
	Source        string   `json:"source,omitempty"`
	ViewportWidth int      `json:"viewport_width,omitempty"`
}

// category migration from WP_REST_Block_Patterns_Controller::$categories_migration.
var patternCategoryMigration = map[string]string{
	"buttons": "call-to-action",
	"columns": "text",
	"query":   "posts",
}

// RESTBlockPatternCategory is a wp/v2/block-patterns/categories item (P4.107).
// Description is always present (empty string when unset) to match the public schema.
type RESTBlockPatternCategory struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

func (s *Server) serveBlockPatternCategories(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P4.106 / P4.107: GET, HEAD, OPTIONS like WP_REST_Block_Pattern_Categories_Controller.
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view",
			"message": "Sorry, you are not allowed to view the registered block pattern categories.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	if r.Method == http.MethodOptions {
		writeJSON(w, http.StatusOK, blockPatternCategorySchema())
		return
	}
	// P4.107: validate context (view|edit|embed); default view.
	ctx := r.URL.Query().Get("context")
	if ctx == "" {
		ctx = "view"
	}
	switch ctx {
	case "view", "edit", "embed":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): context",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{"context": "context is not one of view, edit, embed."},
			},
		})
		return
	}
	if c.PatternCategories == nil {
		c.PatternCategories = wp.NewPatternCategoryRegistry()
		wp.RegisterDefaultPatternCategories(c.PatternCategories, wp.ActiveStylesheet(c))
	}
	cats := c.PatternCategories.All()
	out := make([]RESTBlockPatternCategory, 0, len(cats))
	for _, cat := range cats {
		out = append(out, prepareBlockPatternCategory(cat))
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func prepareBlockPatternCategory(cat wp.PatternCategory) RESTBlockPatternCategory {
	label := cat.Label
	if label == "" {
		label = cat.Name
	}
	return RESTBlockPatternCategory{
		Name:        cat.Name,
		Label:       label,
		Description: cat.Description, // always emit (may be "")
	}
}

func blockPatternCategorySchema() map[string]any {
	return map[string]any{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"title":   "block-pattern-category",
		"type":    "object",
		"properties": map[string]any{
			"name": map[string]any{
				"description": "The category name.",
				"type":        "string",
				"readonly":    true,
				"context":     []string{"view", "edit", "embed"},
			},
			"label": map[string]any{
				"description": "The category label, in human readable format.",
				"type":        "string",
				"readonly":    true,
				"context":     []string{"view", "edit", "embed"},
			},
			"description": map[string]any{
				"description": "The category description, in human readable format.",
				"type":        "string",
				"readonly":    true,
				"context":     []string{"view", "edit", "embed"},
			},
		},
	}
}

func (s *Server) serveBlockPatterns(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P4.106: accept HEAD like WP_REST_Block_Patterns_Controller.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view",
			"message": "Sorry, you are not allowed to view the registered block patterns.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	_ = wp.EnsurePatterns(c)
	patterns := c.Patterns.All()
	out := make([]RESTBlockPattern, 0, len(patterns))
	for _, p := range patterns {
		out = append(out, restPatternFromWP(p))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	w.Header().Set("Allow", "GET, HEAD")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func restPatternFromWP(p wp.BlockPattern) RESTBlockPattern {
	cats := migratePatternCategories(p.Categories)
	src := p.Source
	if src == "wp_block" {
		src = "theme"
	}
	return RESTBlockPattern{
		Name:          p.Name,
		Title:         p.Title,
		Content:       p.Content,
		Description:   p.Description,
		Inserter:      p.Inserter,
		Categories:    cats,
		Keywords:      p.Keywords,
		BlockTypes:    p.BlockTypes,
		PostTypes:     p.PostTypes,
		TemplateTypes: p.TemplateTypes,
		Source:        src,
		ViewportWidth: p.ViewportWidth,
	}
}

func migratePatternCategories(cats []string) []string {
	if len(cats) == 0 {
		return cats
	}
	out := make([]string, len(cats))
	for i, c := range cats {
		if next, ok := patternCategoryMigration[c]; ok {
			out[i] = next
		} else {
			out[i] = c
		}
	}
	return out
}
