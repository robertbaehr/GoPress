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
	// P8.1: fall back to Application Password Basic auth when no cookie session.
	if c.CurrentUser == nil {
		c.AuthenticateApplicationPassword()
	}
	setAPILinkHeader(w, c)

	path := restPath(r.URL.Path)
	switch {
	case path == "/" || path == "":
		s.serveIndex(w, c)
	case path == "/oembed/1.0" || path == "/oembed/1.0/" || path == "/oembed/1.0/embed":
		s.serveOEmbed(w, r, c)
	case path == "/wp/v2/types":
		s.serveTypes(w, r, c)
	case path == "/wp/v2/posts":
		s.servePosts(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/posts/"):
		if parentID, autoID, ok := parsePostAutosavePath(path); ok {
			if autoID > 0 {
				s.servePostAutosaveItem(w, r, c, parentID, autoID)
			} else {
				s.servePostAutosaves(w, r, c, parentID)
			}
			return
		}
		if parentID, revID, ok := parsePostRevisionPath(path); ok {
			if revID > 0 {
				s.servePostRevisionItem(w, r, c, parentID, revID)
			} else {
				s.servePostRevisions(w, r, c, parentID)
			}
			return
		}
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
	case path == "/wp/v2/taxonomies":
		s.serveTaxonomies(w, r, c)
	case path == "/wp/v2/categories":
		s.serveCategories(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/categories/"):
		id, ok := parseTrailingID(path, "/wp/v2/categories/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveCategoryItem(w, r, c, id)
	case path == "/wp/v2/tags":
		s.serveTags(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/tags/"):
		id, ok := parseTrailingID(path, "/wp/v2/tags/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveTagItem(w, r, c, id)
	case path == "/wp/v2/comments":
		s.serveComments(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/comments/"):
		id, ok := parseTrailingID(path, "/wp/v2/comments/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveCommentItem(w, r, c, id)
	case path == "/wp/v2/media":
		s.serveMedia(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/media/"):
		id, ok := parseTrailingID(path, "/wp/v2/media/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveMediaItem(w, r, c, id)
	case path == "/wp/v2/search":
		s.serveSearch(w, r, c)
	case path == "/wp/v2/blocks":
		s.serveBlocks(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/blocks/"):
		id, ok := parseTrailingID(path, "/wp/v2/blocks/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveBlockItem(w, r, c, id)
	case path == "/wp/v2/block-types":
		s.serveBlockTypes(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/block-renderer/"):
		name := strings.TrimPrefix(path, "/wp/v2/block-renderer/")
		name = strings.Trim(name, "/")
		if name == "" || strings.Count(name, "/") != 1 {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveBlockRenderer(w, r, c, name)
	case path == "/wp/v2/navigation":
		s.serveNavigation(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/navigation/"):
		id, ok := parseTrailingID(path, "/wp/v2/navigation/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveNavigationItem(w, r, c, id)
	case path == "/wp/v2/menus":
		s.serveMenus(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/menus/"):
		id, ok := parseTrailingID(path, "/wp/v2/menus/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveMenuItem(w, r, c, id)
	case path == "/wp/v2/menu-items":
		s.serveMenuItems(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/menu-items/"):
		id, ok := parseTrailingID(path, "/wp/v2/menu-items/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveMenuItemByID(w, r, c, id)
	case path == "/wp/v2/menu-locations":
		s.serveMenuLocations(w, r, c)
	case path == "/wp/v2/widgets":
		s.serveWidgets(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/widgets/"):
		id := strings.Trim(strings.TrimPrefix(path, "/wp/v2/widgets/"), "/")
		if id == "" || strings.Contains(id, "/") {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveWidgetItem(w, r, c, id)
	case path == "/wp/v2/widget-types":
		s.serveWidgetTypes(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/widget-types/"):
		id := strings.Trim(strings.TrimPrefix(path, "/wp/v2/widget-types/"), "/")
		if id == "" || strings.Contains(id, "/") {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveWidgetTypeItem(w, r, c, id)
	case path == "/wp/v2/sidebars":
		s.serveSidebars(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/sidebars/"):
		id := strings.Trim(strings.TrimPrefix(path, "/wp/v2/sidebars/"), "/")
		if id == "" || strings.Contains(id, "/") {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveSidebarItem(w, r, c, id)
	case strings.HasPrefix(path, "/wp/v2/global-styles/"):
		kind, stylesheet, id, ok := parseGlobalStylesPath(path)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		if kind == "theme" {
			s.serveGlobalStylesTheme(w, r, c, stylesheet)
			return
		}
		s.serveGlobalStylesItem(w, r, c, id)
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

// parsePostRevisionPath matches /wp/v2/posts/{parent}/revisions[/id].
func parsePostRevisionPath(path string) (parentID, revID int64, ok bool) {
	return parsePostChildPath(path, "revisions")
}

// parsePostAutosavePath matches /wp/v2/posts/{parent}/autosaves[/id].
func parsePostAutosavePath(path string) (parentID, autoID int64, ok bool) {
	return parsePostChildPath(path, "autosaves")
}

func parsePostChildPath(path, child string) (parentID, childID int64, ok bool) {
	rest := strings.TrimPrefix(path, "/wp/v2/posts/")
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[1] != child {
		return 0, 0, false
	}
	parentID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || parentID <= 0 {
		return 0, 0, false
	}
	if len(parts) == 2 {
		return parentID, 0, true
	}
	if len(parts) != 3 {
		return 0, 0, false
	}
	childID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil || childID <= 0 {
		return 0, 0, false
	}
	return parentID, childID, true
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
	// WP rest_send_allow_header: anonymous-readable routes advertise GET.
	if status == http.StatusOK && w.Header().Get("Allow") == "" {
		w.Header().Set("Allow", "GET")
	}
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
	GMTOffset      float64               `json:"gmt_offset"`
	TimezoneString string                `json:"timezone_string"`
	Namespaces     []string              `json:"namespaces"`
	Authentication map[string]any        `json:"authentication"`
	Routes         map[string]RouteIndex `json:"routes"`
	SiteLogo       int64                 `json:"site_logo"`
	SiteIcon       int64                 `json:"site_icon"`
	SiteIconURL    string                `json:"site_icon_url"`
	PageForPosts   int64                 `json:"page_for_posts"`
	PageOnFront    int64                 `json:"page_on_front"`
	ShowOnFront    string                `json:"show_on_front"`
	Links          map[string]any        `json:"_links"`
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
		"/oembed/1.0": {
			Namespace: "oembed/1.0",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"context":   map[string]any{"default": "view", "required": false},
					"namespace": map[string]any{"default": "oembed/1.0", "required": false},
				},
			}},
		},
		"/oembed/1.0/embed": {
			Namespace: "oembed/1.0",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"url":      map[string]any{"required": true, "type": "string", "format": "uri"},
					"format":   map[string]any{"default": "json", "required": false},
					"maxwidth": map[string]any{"default": 600, "required": false},
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
		"/wp/v2/posts/(?P<parent>[\\d]+)/revisions": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		},
		"/wp/v2/posts/(?P<parent>[\\d]+)/revisions/(?P<id>[\\d]+)": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		},
		// WP registers the autosaves collection under (?P<id>) (not parent).
		"/wp/v2/posts/(?P<id>[\\d]+)/autosaves": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		},
		"/wp/v2/posts/(?P<parent>[\\d]+)/autosaves/(?P<id>[\\d]+)": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		},
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
		"/wp/v2/taxonomies": {
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
		"/wp/v2/categories":                collectionRoute("wp/v2"),
		"/wp/v2/categories/(?P<id>[\\d]+)": itemRoute("wp/v2"),
		"/wp/v2/tags":                      collectionRoute("wp/v2"),
		"/wp/v2/tags/(?P<id>[\\d]+)":       itemRoute("wp/v2"),
		"/wp/v2/comments":                  collectionRoute("wp/v2"),
		"/wp/v2/comments/(?P<id>[\\d]+)":   itemRoute("wp/v2"),
		"/wp/v2/media":                     collectionRoute("wp/v2"),
		"/wp/v2/media/(?P<id>[\\d]+)":      itemRoute("wp/v2"),
		"/wp/v2/search": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"context":  map[string]any{"default": "view", "required": false},
					"page":     map[string]any{"default": 1, "required": false},
					"per_page": map[string]any{"default": 10, "required": false},
					"search":   map[string]any{"required": false},
					"type":     map[string]any{"required": false},
					"subtype":  map[string]any{"required": false},
				},
			}},
		},
		"/wp/v2/blocks":                collectionRoute("wp/v2"),
		"/wp/v2/blocks/(?P<id>[\\d]+)": itemRoute("wp/v2"),
		"/wp/v2/block-types": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"context":   map[string]any{"default": "view", "required": false},
					"namespace": map[string]any{"required": false},
				},
			}},
		},
		"/wp/v2/block-renderer/(?P<name>[a-z0-9-]+/[a-z0-9-]+)": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET", "POST"},
				Args: map[string]any{
					"name":       map[string]any{"required": false},
					"attributes": map[string]any{"required": false},
					"post_id":    map[string]any{"required": false},
				},
			}},
		},
		// Auth-gated handlers are still advertised on the public index (WP parity).
		"/wp/v2/settings": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST", "PUT", "PATCH"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
			}},
		},
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

	// Advertise additional public route keys present in WP (handlers may 404 until ported).
	for k, v := range extraIndexRoutes() {
		if _, ok := routes[k]; !ok {
			routes[k] = v
		}
	}

	return Index{
		Name:           name,
		Description:    desc,
		URL:            siteURL,
		Home:           home,
		GMTOffset:      optionFloat(c, "gmt_offset", 0),
		TimezoneString: optionString(c, "timezone_string", ""),
		Namespaces:     namespaces,
		Authentication: auth,
		Routes:         routes,
		SiteLogo:       optionInt64(c, "site_logo", 0),
		SiteIcon:       optionInt64(c, "site_icon", 0),
		SiteIconURL:    "",
		PageForPosts:   optionInt64(c, "page_for_posts", 0),
		PageOnFront:    optionInt64(c, "page_on_front", 0),
		ShowOnFront:    optionString(c, "show_on_front", "posts"),
		Links: map[string]any{
			"help": []map[string]any{{"href": "https://developer.wordpress.org/rest-api/"}},
		},
	}
}

func optionFloat(c *wp.Context, name string, fallback float64) float64 {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, err := strconv.ParseFloat(t, 64)
		if err == nil {
			return f
		}
	}
	return fallback
}

func optionInt64(c *wp.Context, name string, fallback int64) int64 {
	if c == nil || c.Options == nil {
		return fallback
	}
	v := c.Options.GetOption(name, fallback)
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			return n
		}
	}
	return fallback
}

func extraIndexRoutes() map[string]RouteIndex {
	// Middle-path index expansion: register more WP route keys with lean args.
	ns := "wp/v2"
	getOnly := func(namespace string) RouteIndex {
		return RouteIndex{
			Namespace: namespace,
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		}
	}
	return map[string]RouteIndex{
		"/batch/v1":          getOnly("batch/v1"),
		"/oembed/1.0/proxy":  getOnly("oembed/1.0"),
		"/wp-site-health/v1": getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/background-updates":   getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/loopback-requests":    getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/https-status":         getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/dotorg-communication": getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/authorization-header": getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/directory-sizes":            getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/page-cache":           getOnly("wp-site-health/v1"),
		"/wp-block-editor/v1":                           getOnly("wp-block-editor/v1"),
		"/wp-block-editor/v1/export":                    getOnly("wp-block-editor/v1"),
		"/wp-block-editor/v1/url-details":               getOnly("wp-block-editor/v1"),
		"/wp-block-editor/v1/navigation-fallback":       getOnly("wp-block-editor/v1"),
		"/wp-abilities/v1":                              getOnly("wp-abilities/v1"),
		"/wp-abilities/v1/abilities":                    getOnly("wp-abilities/v1"),
		"/wp-abilities/v1/categories":                   getOnly("wp-abilities/v1"),
		"/wp/v2/users":                                  collectionRoute(ns),
		"/wp/v2/users/(?P<id>[\\d]+)":                   itemRoute(ns),
		"/wp/v2/themes":                                 getOnly(ns),
		"/wp/v2/plugins":                                collectionRoute(ns),
		"/wp/v2/menu-items":                             collectionRoute(ns),
		"/wp/v2/menu-items/(?P<id>[\\d]+)":              itemRoute(ns),
		"/wp/v2/menus":                                  collectionRoute(ns),
		"/wp/v2/menus/(?P<id>[\\d]+)":                   itemRoute(ns),
		"/wp/v2/menu-locations":                         getOnly(ns),
		"/wp/v2/templates":                              collectionRoute(ns),
		"/wp/v2/templates/(?P<id>([^\\/:<>\\*\\?\"\\|]+(?:\\/[^\\/:<>\\*\\?\"\\|]+)?)[\\/\\w%-]+)": itemRoute(ns),
		"/wp/v2/template-parts": collectionRoute(ns),
		"/wp/v2/template-parts/(?P<id>([^\\/:<>\\*\\?\"\\|]+(?:\\/[^\\/:<>\\*\\?\"\\|]+)?)[\\/\\w%-]+)": itemRoute(ns),
		"/wp/v2/navigation":                       collectionRoute(ns),
		"/wp/v2/navigation/(?P<id>[\\d]+)":        itemRoute(ns),
		"/wp/v2/global-styles/(?P<id>[\\/\\d+]+)": getOnly(ns),
		"/wp/v2/global-styles/themes/(?P<stylesheet>[\\/\\s%\\w\\.\\(\\)\\[\\]\\@_\\-]+)/variations":     getOnly(ns),
		"/wp/v2/global-styles/themes/(?P<stylesheet>[^\\/:<>\\*\\?\"\\|]+(?:\\/[^\\/:<>\\*\\?\"\\|]+)?)": getOnly(ns),
		"/wp/v2/block-patterns/patterns":             getOnly(ns),
		"/wp/v2/block-patterns/categories":           getOnly(ns),
		"/wp/v2/font-families":                       collectionRoute(ns),
		"/wp/v2/font-families/(?P<id>[\\d]+)":        itemRoute(ns),
		"/wp/v2/types/(?P<type>[\\w-]+)":             getOnly(ns),
		"/wp/v2/statuses":                            getOnly(ns),
		"/wp/v2/statuses/(?P<status>[\\w-]+)":        getOnly(ns),
		"/wp/v2/taxonomies/(?P<taxonomy>[\\w-]+)":    getOnly(ns),
		"/wp/v2/sidebars":                            getOnly(ns),
		"/wp/v2/sidebars/(?P<id>[\\w-]+)":            getOnly(ns),
		"/wp/v2/widgets":                             getOnly(ns),
		"/wp/v2/widgets/(?P<id>[\\w\\-]+)":           getOnly(ns),
		"/wp/v2/widget-types":                        getOnly(ns),
		"/wp/v2/widget-types/(?P<id>[a-zA-Z0-9_-]+)": getOnly(ns),
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
