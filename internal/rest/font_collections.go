package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// FontCollectionResponse is the REST shape for wp/v2/font-collections (P4.55).
type FontCollectionResponse struct {
	Slug         string                      `json:"slug"`
	Name         string                      `json:"name"`
	Description  string                      `json:"description"`
	FontFamilies []any                       `json:"font_families"`
	Categories   []wp.FontCollectionCategory `json:"categories"`
	Links        map[string]any              `json:"_links,omitempty"`
}

func canReadFontCollections(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, "edit_theme_options")
}

func (s *Server) serveFontCollections(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canReadFontCollections(c) {
		writeFontsForbidden(w, c, "Sorry, you are not allowed to access font collections.")
		return
	}
	lib := c.FontLibrary
	if lib == nil {
		lib = wp.NewFontLibrary()
	}
	all := lib.All()
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	total := len(all)
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}
	slice := all[start:end]
	out := make([]FontCollectionResponse, 0, len(slice))
	for _, col := range slice {
		out = append(out, fontCollectionResponse(col))
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	w.Header().Set("Allow", "GET")
	w.Header().Set("X-GoPress-Font-Collections", "remote-families-skipped")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveFontCollectionItem(w http.ResponseWriter, r *http.Request, c *wp.Context, slug string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canReadFontCollections(c) {
		writeFontsForbidden(w, c, "Sorry, you are not allowed to access font collections.")
		return
	}
	slug = strings.Trim(slug, "/")
	if slug == "" || strings.Contains(slug, "/") {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	lib := c.FontLibrary
	if lib == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_font_collection_not_found",
			"message": "Font collection not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	col, ok := lib.Get(slug)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_font_collection_not_found",
			"message": "Font collection not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	w.Header().Set("Allow", "GET")
	w.Header().Set("X-GoPress-Font-Collections", "remote-families-skipped")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, fontCollectionResponse(col))
}

func fontCollectionResponse(col wp.FontCollection) FontCollectionResponse {
	families := col.FontFamilies
	if families == nil {
		families = []any{}
	}
	cats := col.Categories
	if cats == nil {
		cats = []wp.FontCollectionCategory{}
	}
	return FontCollectionResponse{
		Slug:         col.Slug,
		Name:         col.Name,
		Description:  col.Description,
		FontFamilies: families,
		Categories:   cats,
		Links: map[string]any{
			"self": []map[string]any{
				{"href": "/wp-json/wp/v2/font-collections/" + col.Slug},
			},
			"collection": []map[string]any{
				{"href": "/wp-json/wp/v2/font-collections"},
			},
		},
	}
}
