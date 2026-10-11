package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// knownRESTNamespaces are namespaces advertised on the root index (P4.127).
var knownRESTNamespaces = []string{
	"oembed/1.0",
	"wp/v2",
	"wp-site-health/v1",
	"wp-block-editor/v1",
	"wp-abilities/v1",
}

// matchNamespaceIndex returns the namespace when path is exactly /{namespace}[/].
func matchNamespaceIndex(path string) (string, bool) {
	p := strings.Trim(path, "/")
	if p == "" {
		return "", false
	}
	for _, ns := range knownRESTNamespaces {
		if p == ns {
			return ns, true
		}
	}
	return "", false
}

// serveNamespaceIndex mirrors WP_REST_Server::get_namespace_index (P4.127).
func (s *Server) serveNamespaceIndex(w http.ResponseWriter, r *http.Request, c *wp.Context, namespace string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_invalid_namespace",
			"message": "The specified namespace could not be found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}

	routes := namespaceRoutes(c, namespace)
	if len(routes) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_invalid_namespace",
			"message": "The specified namespace could not be found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}

	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	up := home + "/wp-json/"
	if home == "" {
		up = "/wp-json/"
	}

	payload := map[string]any{
		"namespace": namespace,
		"routes":    routes,
		"_links": map[string]any{
			"up": []map[string]any{{"href": up}},
		},
	}
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("rest_namespace_index", payload, namespace)
		if m, ok := filtered.(map[string]any); ok && m != nil {
			payload = m
		}
	}

	w.Header().Set("Allow", "GET")
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func namespaceRoutes(c *wp.Context, namespace string) map[string]RouteIndex {
	out := map[string]RouteIndex{}
	if golden, err := loadIndexRoutesAnon(); err == nil {
		for k, v := range golden {
			if v.Namespace == namespace || strings.HasPrefix(strings.TrimPrefix(k, "/"), namespace) {
				out[k] = v
			}
		}
	}
	if len(out) == 0 {
		for k, v := range builtInIndexRoutes() {
			if v.Namespace == namespace {
				out[k] = v
			}
		}
		for k, v := range extraIndexRoutes() {
			if v.Namespace == namespace {
				if _, ok := out[k]; !ok {
					out[k] = v
				}
			}
		}
	}
	if c != nil && c.CurrentUser != nil {
		for k, v := range adminExtraRoutes() {
			if v.Namespace == namespace {
				if _, ok := out[k]; !ok {
					out[k] = v
				}
			}
		}
	}
	// Ensure the namespace root route is present.
	root := "/" + namespace
	if _, ok := out[root]; !ok {
		out[root] = RouteIndex{
			Namespace: namespace,
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		}
	}
	return out
}
