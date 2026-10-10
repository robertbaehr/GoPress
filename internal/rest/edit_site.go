package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// NavigationFallback is the wp-block-editor/v1/navigation-fallback body (P4.25/P4.53).
type NavigationFallback struct {
	ID    int64          `json:"id"`
	Links map[string]any `json:"_links,omitempty"`
}

func (s *Server) serveEditSiteExport(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if c == nil || c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "export") {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_export_templates",
			"message": "Sorry, you are not allowed to export templates and template parts.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	// Full theme ZIP export lands later; advertise the route and gate auth now.
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"code":    "rest_export_not_implemented",
		"message": "GoPress edit-site export stub — ZIP export lands later.",
		"data":    map[string]any{"status": 501},
	})
}

func (s *Server) serveNavigationFallback(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if c == nil || c.CurrentUser == nil || !canEditPosts(c) || !c.UserCan(c.CurrentUser, "edit_theme_options") {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_create",
			"message": "Sorry, you are not allowed to create Navigation Menus as this user.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	post, err := wp.GetOrCreateNavigationFallback(c)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_navigation_fallback_error",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "no_fallback_menu",
			"message": "No fallback menu found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, NavigationFallback{
		ID: post.ID,
		Links: map[string]any{
			"self": []map[string]any{{
				"href":       home + "/wp-json/wp/v2/navigation/" + strconv.FormatInt(post.ID, 10),
				"embeddable": true,
			}},
		},
	})
}
