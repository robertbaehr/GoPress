package rest

import (
	"net/http"
	"sort"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// RESTBlockPattern is a wp/v2/block-patterns/patterns item (P5.38).
// Field names follow the WP REST schema (snake_case).
type RESTBlockPattern struct {
	Name          string   `json:"name"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Description   string   `json:"description,omitempty"`
	Inserter      bool     `json:"inserter"`
	Categories    []string `json:"categories,omitempty"`
	BlockTypes    []string `json:"block_types,omitempty"`
	Source        string   `json:"source,omitempty"`
	ViewportWidth int      `json:"viewport_width,omitempty"`
}

// category migration from WP_REST_Block_Patterns_Controller::$categories_migration.
var patternCategoryMigration = map[string]string{
	"buttons": "call-to-action",
	"columns": "text",
	"query":   "posts",
}

func (s *Server) serveBlockPatternCategories(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
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
	if c.PatternCategories == nil {
		c.PatternCategories = wp.NewPatternCategoryRegistry()
		wp.RegisterDefaultPatternCategories(c.PatternCategories, wp.ActiveStylesheet(c))
	}
	cats := c.PatternCategories.All()
	out := make([]wp.PatternCategory, 0, len(cats))
	out = append(out, cats...)
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveBlockPatterns(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
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
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, out)
}

func restPatternFromWP(p wp.BlockPattern) RESTBlockPattern {
	cats := migratePatternCategories(p.Categories)
	src := p.Source
	if src == "wp_block" {
		src = "theme"
	}
	return RESTBlockPattern{
		Name:        p.Name,
		Title:       p.Title,
		Content:     p.Content,
		Description: p.Description,
		Inserter:    p.Inserter,
		Categories:  cats,
		BlockTypes:  p.BlockTypes,
		Source:      src,
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
