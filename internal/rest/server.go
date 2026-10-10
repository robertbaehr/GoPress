// Package rest is a minimal WordPress REST API server (P4.1+).
package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Server serves /wp-json routes.
type Server struct{}

// Handle dispatches a request whose path is /wp-json or under /wp-json/.
func (s *Server) Handle(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if c == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"code":    "rest_unavailable",
			"message": "GoPress REST API unavailable",
		})
		return
	}
	c.AuthenticateRequest()

	path := restPath(r.URL.Path)
	switch {
	case path == "/" || path == "":
		s.serveIndex(w, c)
	case path == "/wp/v2/types":
		s.serveTypes(w, r, c)
	case path == "/wp/v2/posts":
		s.servePosts(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/posts/"):
		id, ok := parseTrailingID(path, "/wp/v2/posts/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.servePostItem(w, r, c, id)
	case path == "/wp/v2/pages":
		s.servePages(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/pages/"):
		id, ok := parseTrailingID(path, "/wp/v2/pages/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.servePageItem(w, r, c, id)
	case path == "/wp/v2/settings":
		s.serveSettings(w, r, c)
	case path == "/wp/v2/users/me":
		s.serveUsersMe(w, r, c)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 404},
		})
	}
}

func parseTrailingID(path, prefix string) (int64, bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" || strings.Contains(rest, "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func restPath(urlPath string) string {
	p := urlPath
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimPrefix(p, "/wp-json")
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func (s *Server) serveIndex(w http.ResponseWriter, c *wp.Context) {
	idx := buildIndex(c)
	writeJSON(w, http.StatusOK, idx)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// Index is the GET /wp-json/ response body (WP_REST_Server::get_index subset).
type Index struct {
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	URL            string                `json:"url"`
	Home           string                `json:"home"`
	Namespaces     []string              `json:"namespaces"`
	Authentication map[string]any        `json:"authentication"`
	Routes         map[string]RouteIndex `json:"routes"`
}

// RouteIndex describes one registered route in the index.
type RouteIndex struct {
	Namespace string          `json:"namespace"`
	Methods   []string        `json:"methods"`
	Endpoints []RouteEndpoint `json:"endpoints"`
}

// RouteEndpoint is a method group under a route.
type RouteEndpoint struct {
	Methods []string       `json:"methods"`
	Args    map[string]any `json:"args,omitempty"`
}

func buildIndex(c *wp.Context) Index {
	name := optionString(c, "blogname", "")
	desc := optionString(c, "blogdescription", "")
	siteURL := strings.TrimRight(optionString(c, "siteurl", ""), "/")
	home := strings.TrimRight(optionString(c, "home", siteURL), "/")
	if home == "" {
		home = siteURL
	}

	namespaces := []string{
		"oembed/1.0",
		"wp/v2",
		"wp-site-health/v1",
		"wp-block-editor/v1",
		"wp-abilities/v1",
	}

	routes := map[string]RouteIndex{
		"/": {
			Namespace: "",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"context": map[string]any{"default": "view", "required": false},
				},
			}},
		},
		"/wp/v2": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"namespace": map[string]any{"default": "wp/v2", "required": false},
					"context":   map[string]any{"default": "view", "required": false},
				},
			}},
		},
		"/wp/v2/posts":                collectionRoute("wp/v2"),
		"/wp/v2/posts/(?P<id>[\\d]+)": itemRoute("wp/v2"),
		"/wp/v2/types": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"context": map[string]any{
						"default":  "view",
						"required": false,
						"enum":     []string{"view", "embed", "edit"},
					},
				},
			}},
		},
		"/wp/v2/pages":                collectionRoute("wp/v2"),
		"/wp/v2/pages/(?P<id>[\\d]+)": itemRoute("wp/v2"),
	}

	// When authenticated, advertise additional admin-oriented routes (best-effort).
	if c != nil && c.CurrentUser != nil {
		for k, v := range adminExtraRoutes() {
			routes[k] = v
		}
	}

	auth := map[string]any{}
	if siteURL != "" {
		auth["application-passwords"] = map[string]any{
			"endpoints": map[string]any{
				"authorization": siteURL + "/wp-admin/authorize-application.php",
			},
		}
	}

	return Index{
		Name:           name,
		Description:    desc,
		URL:            siteURL,
		Home:           home,
		Namespaces:     namespaces,
		Authentication: auth,
		Routes:         routes,
	}
}

func collectionRoute(ns string) RouteIndex {
	return RouteIndex{
		Namespace: ns,
		Methods:   []string{"GET", "POST"},
		Endpoints: []RouteEndpoint{
			{
				Methods: []string{"GET"},
				Args: map[string]any{
					"context":  map[string]any{"default": "view", "required": false},
					"page":     map[string]any{"default": 1, "required": false},
					"per_page": map[string]any{"default": 10, "required": false},
					"search":   map[string]any{"required": false},
					"slug":     map[string]any{"required": false},
				},
			},
			{
				Methods: []string{"POST"},
				Args: map[string]any{
					"title":   map[string]any{"required": false},
					"content": map[string]any{"required": false},
					"status":  map[string]any{"required": false},
				},
			},
		},
	}
}

func itemRoute(ns string) RouteIndex {
	return RouteIndex{
		Namespace: ns,
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []RouteEndpoint{{
			Methods: []string{"GET"},
			Args: map[string]any{
				"id":      map[string]any{"required": false},
				"context": map[string]any{"default": "view", "required": false},
			},
		}},
	}
}

func adminExtraRoutes() map[string]RouteIndex {
	return map[string]RouteIndex{
		"/wp/v2/users/me": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"context": map[string]any{"default": "view", "required": false},
				},
			}},
		},
		"/wp/v2/settings": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST", "PUT", "PATCH"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
			}},
		},
		"/wp/v2/plugins": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
			}},
		},
		"/wp/v2/themes": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
			}},
		},
		"/wp/v2/users": collectionRoute("wp/v2"),
		"/wp/v2/media": collectionRoute("wp/v2"),
	}
}

func optionString(c *wp.Context, name, fallback string) string {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return fallback
	default:
		s := fmt.Sprint(t)
		if s == "" || s == "<nil>" {
			return fallback
		}
		return s
	}
}
