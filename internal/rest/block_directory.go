package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveBlockDirectorySearch stubs the wordpress.org block directory proxy (P4.42 / P4.101).
// Remote fetches to api.wordpress.org are out of scope and return 403 when the
// caller would otherwise be allowed to browse the directory.
//
// P4.101: require non-empty `term` (collection param required + minLength 1)
// after the browse capability gate, matching WP_REST_Block_Directory_Controller.
func (s *Server) serveBlockDirectorySearch(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	// WP requires install_plugins+activate_plugins; GoPress denies those caps for
	// parity, so treat a logged-in editor as the browse gate and always refuse
	// the remote wordpress.org lookup with 403.
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_block_directory_cannot_view",
			"message": "Sorry, you are not allowed to browse the block directory.",
			"data":    map[string]any{"status": http.StatusUnauthorized},
		})
		return
	}
	if !canEditPosts(c) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_block_directory_cannot_view",
			"message": "Sorry, you are not allowed to browse the block directory.",
			"data":    map[string]any{"status": http.StatusForbidden},
		})
		return
	}

	// P4.101: term is required (minLength 1). Check presence vs empty separately
	// so missing and empty map to WP-shaped error codes.
	if _, ok := r.URL.Query()["term"]; !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_missing_callback_param",
			"message": "Missing parameter(s): term",
			"data": map[string]any{
				"status": 400,
				"params": []string{"term"},
			},
		})
		return
	}
	term := strings.TrimSpace(r.URL.Query().Get("term"))
	if term == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): term",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{
					"term": "term must be a non-empty string.",
				},
			},
		})
		return
	}

	// Accept page/per_page like the collection schema (ignored; remote blocked).
	_ = r.URL.Query().Get("page")
	_ = r.URL.Query().Get("per_page")

	// Remote directory blocked: never call api.wordpress.org / plugins_api.
	w.Header().Set("Allow", "GET")
	w.Header().Set("X-GoPress-Block-Directory", "remote-blocked")
	writeJSON(w, http.StatusForbidden, map[string]any{
		"code":    "rest_block_directory_remote_disabled",
		"message": "The remote block directory is out of scope for GoPress.",
		"data":    map[string]any{"status": 403, "term": term},
	})
}
