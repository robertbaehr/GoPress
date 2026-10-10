package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// SiteHealthTest is a wp-site-health/v1 test result stub (P8.3 / P4.39).
type SiteHealthTest struct {
	Test        string         `json:"test"`
	Label       string         `json:"label"`
	Status      string         `json:"status"`
	Badge       map[string]any `json:"badge"`
	Description string         `json:"description"`
	Actions     string         `json:"actions"`
}

// knownSiteHealthTests lists registered async Site Health REST tests (P4.39).
var knownSiteHealthTests = []string{
	"background-updates",
	"loopback-requests",
	"https-status",
	"dotorg-communication",
	"authorization-header",
	"page-cache",
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
		routes := make([]string, 0, len(knownSiteHealthTests)+2)
		routes = append(routes, "/wp-site-health/v1/tests")
		for _, name := range knownSiteHealthTests {
			routes = append(routes, "/wp-site-health/v1/tests/"+name)
		}
		routes = append(routes, "/wp-site-health/v1/directory-sizes")
		writeJSON(w, http.StatusOK, map[string]any{
			"namespace": "wp-site-health/v1",
			"routes":    routes,
		})
		return
	case "/wp-site-health/v1/tests", "/wp-site-health/v1/tests/":
		// Collection index of registered async tests (P4.39).
		w.Header().Set("Allow", "GET")
		tests := make([]SiteHealthTest, 0, len(knownSiteHealthTests))
		for _, name := range knownSiteHealthTests {
			tests = append(tests, stubSiteHealthTest(name))
		}
		writeJSON(w, http.StatusOK, tests)
		return
	case "/wp-site-health/v1/directory-sizes":
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, map[string]any{
			"wordpress_size": map[string]any{"size": "0 B", "debug": "0 B (0 bytes)", "raw": 0},
			"themes_size":    map[string]any{"size": "0 B", "debug": "0 B (0 bytes)", "raw": 0},
			"plugins_size":   map[string]any{"size": "0 B", "debug": "0 B (0 bytes)", "raw": 0},
			"uploads_size":   map[string]any{"size": "0 B", "debug": "0 B (0 bytes)", "raw": 0},
			"database_size":  map[string]any{"size": "0 B", "debug": "0 B (0 bytes)", "raw": 0},
			"total_size":     map[string]any{"size": "0 B", "debug": "0 B (0 bytes)", "raw": 0},
			"raw":            0,
		})
		return
	}

	if strings.HasPrefix(path, "/wp-site-health/v1/tests/") {
		name := strings.Trim(strings.TrimPrefix(path, "/wp-site-health/v1/tests/"), "/")
		if name == "" || strings.Contains(name, "/") || !isKnownSiteHealthTest(name) {
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

func isKnownSiteHealthTest(name string) bool {
	for _, t := range knownSiteHealthTests {
		if t == name {
			return true
		}
	}
	return false
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
	meta := map[string]struct {
		label string
		badge string
		color string
	}{
		"background-updates":   {"Background updates", "Security", "blue"},
		"loopback-requests":    {"Loopback request", "Performance", "blue"},
		"https-status":         {"HTTPS status", "Security", "blue"},
		"dotorg-communication": {"Communication with WordPress.org", "Security", "blue"},
		"authorization-header": {"Authorization header", "Security", "blue"},
		"page-cache":           {"Page cache", "Performance", "blue"},
	}
	m, ok := meta[name]
	if !ok {
		m = struct {
			label string
			badge string
			color string
		}{name, "Performance", "blue"}
	}
	return SiteHealthTest{
		Test:   strings.ReplaceAll(name, "-", "_"),
		Label:  m.label,
		Status: "good",
		Badge: map[string]any{
			"label": m.badge,
			"color": m.color,
		},
		Description: "<p>GoPress site-health stub — full checks land later.</p>",
		Actions:     "",
	}
}
