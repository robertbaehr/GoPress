package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// URLDetails is the wp-block-editor/v1/url-details response (P4.24 stub).
// Remote HTML fetch is intentionally omitted; fields are empty placeholders.
type URLDetails struct {
	Title       string `json:"title"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

func (s *Server) serveURLDetails(w http.ResponseWriter, r *http.Request, c *wp.Context) {
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
			"code":    "rest_cannot_view_url_details",
			"message": "Sorry, you are not allowed to process remote URLs.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_missing_callback_param",
			"message": "Missing parameter(s): url",
			"data": map[string]any{
				"status": 400,
				"params": []string{"url"},
			},
		})
		return
	}
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, URLDetails{})
}
