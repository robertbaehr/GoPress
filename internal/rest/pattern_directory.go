package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// servePatternDirectoryPatterns stubs the wordpress.org pattern directory proxy (P4.37 / P4.105).
// Remote fetches to api.wordpress.org are out of scope; authenticated editors get an
// empty local list, and anonymous callers mirror WP's cannot_view error.
//
// P4.105: accept HEAD; validate search minLength, category/keyword minimums, and
// orderby enum after the browse capability gate (collection params subset).
func (s *Server) servePatternDirectoryPatterns(w http.ResponseWriter, r *http.Request, c *wp.Context) {
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
			"code":    "rest_pattern_directory_cannot_view",
			"message": "Sorry, you are not allowed to browse the local block pattern directory.",
			"data":    map[string]any{"status": status},
		})
		return
	}

	q := r.URL.Query()

	// search minLength 1 when present (WP collection param).
	if _, ok := q["search"]; ok {
		search := strings.TrimSpace(q.Get("search"))
		if search == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): search",
				"data": map[string]any{
					"status": 400,
					"params": map[string]any{
						"search": "search must be a non-empty string.",
					},
				},
			})
			return
		}
	}

	// category / keyword: integer minimum 1 when present.
	for _, key := range []string{"category", "keyword"} {
		if _, ok := q[key]; !ok {
			continue
		}
		raw := strings.TrimSpace(q.Get(key))
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): " + key,
				"data": map[string]any{
					"status": 400,
					"params": map[string]any{
						key: key + " must be an integer greater than or equal to 1.",
					},
				},
			})
			return
		}
	}

	// orderby enum (subset of WP_REST_Pattern_Directory_Controller).
	if _, ok := q["orderby"]; ok {
		orderby := strings.ToLower(strings.TrimSpace(q.Get("orderby")))
		if !patternDirectoryOrderbyOK(orderby) {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): orderby",
				"data": map[string]any{
					"status": 400,
					"params": map[string]any{
						"orderby": "orderby is not one of author, date, id, include, modified, parent, relevance, slug, include_slugs, title, favorite_count.",
					},
				},
			})
			return
		}
	}

	// order enum.
	if _, ok := q["order"]; ok {
		order := strings.ToLower(strings.TrimSpace(q.Get("order")))
		if order != "asc" && order != "desc" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": "Invalid parameter(s): order",
				"data": map[string]any{
					"status": 400,
					"params": map[string]any{
						"order": "order is not one of asc, desc.",
					},
				},
			})
			return
		}
	}

	// Accept page/per_page/offset/slug like the collection schema (ignored; remote blocked).
	_ = q.Get("page")
	_ = q.Get("per_page")
	_ = q.Get("offset")
	_ = q.Get("slug")

	// Remote directory blocked: never call api.wordpress.org.
	w.Header().Set("Allow", "GET")
	w.Header().Set("X-GoPress-Pattern-Directory", "remote-blocked")
	if r.Method == http.MethodHead {
		// WP returns early for HEAD without body (get_items HEAD short-circuit).
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, []any{})
}

func patternDirectoryOrderbyOK(v string) bool {
	switch v {
	case "author", "date", "id", "include", "modified", "parent", "relevance",
		"slug", "include_slugs", "title", "favorite_count":
		return true
	default:
		return false
	}
}
