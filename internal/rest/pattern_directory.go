package rest

import (
	"net/http"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// servePatternDirectoryPatterns stubs the wordpress.org pattern directory proxy (P4.37).
// Remote fetches to api.wordpress.org are out of scope; authenticated editors get an
// empty local list, and anonymous callers mirror WP's cannot_view error.
func (s *Server) servePatternDirectoryPatterns(w http.ResponseWriter, r *http.Request, c *wp.Context) {
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
			"code":    "rest_pattern_directory_cannot_view",
			"message": "Sorry, you are not allowed to browse the local block pattern directory.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	// Remote directory blocked: never call api.wordpress.org.
	w.Header().Set("Allow", "GET")
	w.Header().Set("X-GoPress-Pattern-Directory", "remote-blocked")
	writeJSON(w, http.StatusOK, []any{})
}
