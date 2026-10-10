package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// SiteHealthTest is a wp-site-health/v1 test result stub (P8.3).
type SiteHealthTest struct {
	Test        string         `json:"test"`
	Label       string         `json:"label"`
	Status      string         `json:"status"`
	Badge       map[string]any `json:"badge"`
	Description string         `json:"description"`
	Actions     string         `json:"actions"`
}

func (s *Server) serveSiteHealth(w http.ResponseWriter, r *http.Request, c *wp.Context, path string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewSiteHealth(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_forbidden",
			"message": "Sorry, you are not allowed to access site health.",
			"data":    map[string]any{"status": status},
		})
		return
	}

	switch path {
	case "/wp-site-health/v1", "/wp-site-health/v1/":
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, map[string]any{
			"namespace": "wp-site-health/v1",
			"routes": []string{
				"/wp-site-health/v1/tests/background-updates",
				"/wp-site-health/v1/tests/loopback-requests",
				"/wp-site-health/v1/tests/https-status",
				"/wp-site-health/v1/tests/dotorg-communication",
				"/wp-site-health/v1/tests/authorization-header",
				"/wp-site-health/v1/tests/page-cache",
				"/wp-site-health/v1/directory-sizes",
			},
		})
		return
	case "/wp-site-health/v1/directory-sizes":
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, map[string]any{
			"wordpress_size": map[string]any{"size": "0 B", "raw": 0},
			"themes_size":    map[string]any{"size": "0 B", "raw": 0},
			"plugins_size":   map[string]any{"size": "0 B", "raw": 0},
			"uploads_size":   map[string]any{"size": "0 B", "raw": 0},
			"database_size":  map[string]any{"size": "0 B", "raw": 0},
			"total_size":     map[string]any{"size": "0 B", "raw": 0},
		})
		return
	}

	if strings.HasPrefix(path, "/wp-site-health/v1/tests/") {
		name := strings.Trim(strings.TrimPrefix(path, "/wp-site-health/v1/tests/"), "/")
		if name == "" || strings.Contains(name, "/") {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, stubSiteHealthTest(name))
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]any{
		"code":    "rest_no_route",
		"message": "No route was found matching the URL and request method.",
		"data":    map[string]any{"status": 404},
	})
}

func canViewSiteHealth(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	// WP uses view_site_health_checks; fall back to install_plugins / manage_options.
	if c.UserCan(c.CurrentUser, "view_site_health_checks") {
		return true
	}
	return c.UserCan(c.CurrentUser, "manage_options")
}

func stubSiteHealthTest(name string) SiteHealthTest {
	labels := map[string]string{
		"background-updates":   "Background updates",
		"loopback-requests":    "Loopback requests",
		"https-status":         "HTTPS status",
		"dotorg-communication": "Communication with WordPress.org",
		"authorization-header": "Authorization header",
		"page-cache":           "Page cache",
	}
	label := labels[name]
	if label == "" {
		label = name
	}
	return SiteHealthTest{
		Test:   strings.ReplaceAll(name, "-", "_"),
		Label:  label + " look good (GoPress stub)",
		Status: "good",
		Badge: map[string]any{
			"label": "Performance",
			"color": "blue",
		},
		Description: "<p>GoPress site-health stub — full checks land later.</p>",
		Actions:     "",
	}
}
