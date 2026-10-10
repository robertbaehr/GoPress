// Package rest is a minimal WordPress REST API server (P4.1+).
package rest

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	_ "embed"

	"github.com/robertbaehr/GoPress/internal/wp"
)

//go:embed testdata/index_routes_anon.json.gz
var indexRoutesAnonGZ []byte

var (
	indexRoutesAnonOnce sync.Once
	indexRoutesAnon     map[string]RouteIndex
	indexRoutesAnonErr  error
)

func loadIndexRoutesAnon() (map[string]RouteIndex, error) {
	indexRoutesAnonOnce.Do(func() {
		gr, err := gzip.NewReader(bytes.NewReader(indexRoutesAnonGZ))
		if err != nil {
			indexRoutesAnonErr = err
			return
		}
		defer gr.Close()
		raw, err := io.ReadAll(gr)
		if err != nil {
			indexRoutesAnonErr = err
			return
		}
		var routes map[string]RouteIndex
		if err := json.Unmarshal(raw, &routes); err != nil {
			indexRoutesAnonErr = err
			return
		}
		indexRoutesAnon = routes
	})
	return indexRoutesAnon, indexRoutesAnonErr
}

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

	path := restPathFromRequest(r)
	switch {
	case path == "/" || path == "":
		s.serveIndex(w, c)
	case path == "/batch/v1" || path == "/batch/v1/":
		s.serveBatch(w, r, c)
	case path == "/oembed/1.0" || path == "/oembed/1.0/" || path == "/oembed/1.0/embed":
		s.serveOEmbed(w, r, c)
	case path == "/wp/v2/types":
		s.serveTypes(w, r, c)
	case path == "/wp/v2/statuses":
		s.serveStatuses(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/statuses/"):
		slug := strings.Trim(strings.TrimPrefix(path, "/wp/v2/statuses/"), "/")
		if slug == "" || strings.Contains(slug, "/") {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveStatusItem(w, r, c, slug)
	case path == "/wp/v2/themes":
		s.serveThemes(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/themes/"):
		ss := strings.TrimPrefix(path, "/wp/v2/themes/")
		ss = strings.Trim(ss, "/")
		if ss == "" {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveThemeItem(w, r, c, ss)
	case path == "/wp/v2/plugins":
		s.servePlugins(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/plugins/"):
		plugin := strings.TrimPrefix(path, "/wp/v2/plugins/")
		plugin = strings.Trim(plugin, "/")
		if plugin == "" {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.servePluginItem(w, r, c, plugin)
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
	case path == "/wp/v2/users":
		s.serveUsers(w, r, c)
	case path == "/wp/v2/users/me":
		s.serveUsersMe(w, r, c)
	case strings.Contains(path, "/application-passwords"):
		userKey, uuid, ok := parseApplicationPasswordsPath(path)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveApplicationPasswords(w, r, c, userKey, uuid)
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
	case strings.HasPrefix(path, "/wp/v2/menu-locations/"):
		loc := strings.Trim(strings.TrimPrefix(path, "/wp/v2/menu-locations/"), "/")
		if loc == "" || strings.Contains(loc, "/") {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveMenuLocationItem(w, r, c, loc)
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
	case path == "/wp/v2/font-families" || strings.HasPrefix(path, "/wp/v2/font-families/"):
		familyID, faceID, facesCollection, ok := parseFontFamiliesPath(path)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		if familyID == 0 {
			s.serveFontFamilies(w, r, c)
			return
		}
		if facesCollection {
			if faceID > 0 {
				s.serveFontFaceItem(w, r, c, familyID, faceID)
				return
			}
			s.serveFontFaces(w, r, c, familyID)
			return
		}
		s.serveFontFamilyItem(w, r, c, familyID)
	case path == "/wp/v2/font-collections":
		s.serveFontCollections(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/font-collections/"):
		slug := strings.Trim(strings.TrimPrefix(path, "/wp/v2/font-collections/"), "/")
		s.serveFontCollectionItem(w, r, c, slug)
	case path == "/wp/v2/templates":
		s.serveTemplates(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/templates/"):
		id, ok := parseTemplateIDPath(path, "/wp/v2/templates/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveTemplateItem(w, r, c, id)
	case path == "/wp/v2/template-parts":
		s.serveTemplateParts(w, r, c)
	case strings.HasPrefix(path, "/wp/v2/template-parts/"):
		id, ok := parseTemplateIDPath(path, "/wp/v2/template-parts/")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"code":    "rest_no_route",
				"message": "No route was found matching the URL and request method.",
				"data":    map[string]any{"status": 404},
			})
			return
		}
		s.serveTemplatePartItem(w, r, c, id)
	case path == "/wp/v2/block-patterns/categories":
		s.serveBlockPatternCategories(w, r, c)
	case path == "/wp/v2/block-patterns/patterns":
		s.serveBlockPatterns(w, r, c)
	case path == "/wp/v2/pattern-directory/patterns":
		s.servePatternDirectoryPatterns(w, r, c)
	case path == "/wp/v2/block-directory/search":
		s.serveBlockDirectorySearch(w, r, c)
	case path == "/wp-block-editor/v1/url-details":
		s.serveURLDetails(w, r, c)
	case path == "/wp-block-editor/v1/export":
		s.serveEditSiteExport(w, r, c)
	case path == "/wp-block-editor/v1/navigation-fallback":
		s.serveNavigationFallback(w, r, c)
	case path == "/wp-site-health/v1" || path == "/wp-site-health/v1/" ||
		strings.HasPrefix(path, "/wp-site-health/v1/"):
		s.serveSiteHealth(w, r, c, path)
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

// restPathFromRequest prefers ?rest_route= when set (plain permalinks / index.php, P3.29).
func restPathFromRequest(r *http.Request) string {
	if r != nil && r.URL != nil {
		if rr := strings.TrimSpace(r.URL.Query().Get("rest_route")); rr != "" {
			if !strings.HasPrefix(rr, "/") {
				rr = "/" + rr
			}
			return rr
		}
	}
	if r == nil {
		return "/"
	}
	return restPath(r.URL.Path)
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
	GMTOffset      any                   `json:"gmt_offset"`
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
	Links     any             `json:"_links,omitempty"`
}

// RouteEndpoint is a method group under a route.
type RouteEndpoint struct {
	Methods    []string `json:"methods"`
	Args       any      `json:"args,omitempty"` // WP uses object or empty array
	AllowBatch any      `json:"allow_batch,omitempty"`
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

	// P4.34: prefer embedded WP anon route catalog for index parity.
	routes := map[string]RouteIndex{}
	if golden, err := loadIndexRoutesAnon(); err == nil && len(golden) > 0 {
		for k, v := range golden {
			routes[k] = v
		}
	} else {
		routes = builtInIndexRoutes()
		for k, v := range extraIndexRoutes() {
			if _, ok := routes[k]; !ok {
				routes[k] = v
			}
		}
	}

	// When authenticated, advertise additional admin-oriented routes (best-effort).
	if c != nil && c.CurrentUser != nil {
		for k, v := range adminExtraRoutes() {
			if _, ok := routes[k]; !ok {
				routes[k] = v
			}
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
		GMTOffset:      optionGMTOffset(c),
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

// optionGMTOffset mirrors WP's rest index encoding (string when option is string).
func optionGMTOffset(c *wp.Context) any {
	if c == nil || c.Options == nil {
		return "0"
	}
	v := c.Options.GetOption("gmt_offset", "0")
	switch t := v.(type) {
	case string:
		if t == "" {
			return "0"
		}
		return t
	case float64, int, int64:
		return t
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		if s == "" {
			return "0"
		}
		return s
	}
}

func builtInIndexRoutes() map[string]RouteIndex {
	return map[string]RouteIndex{
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
		"/wp/v2/pages":                     collectionRoute("wp/v2"),
		"/wp/v2/pages/(?P<id>[\\d]+)":      itemRoute("wp/v2"),
		"/wp/v2/taxonomies":                {Namespace: "wp/v2", Methods: []string{"GET"}, Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}}},
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
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		},
		"/wp/v2/blocks":                collectionRoute("wp/v2"),
		"/wp/v2/blocks/(?P<id>[\\d]+)": itemRoute("wp/v2"),
		"/wp/v2/block-types": {
			Namespace: "wp/v2",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		},
		"/wp/v2/block-renderer/(?P<name>[a-z0-9-]+/[a-z0-9-]+)": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET", "POST"}}},
		},
		"/wp/v2/settings": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST", "PUT", "PATCH"},
			Endpoints: []RouteEndpoint{
				{Methods: []string{"GET"}},
				{Methods: []string{"POST", "PUT", "PATCH"}},
			},
		},
		"/wp/v2/users/me": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		},
		"/wp/v2/users/(?P<user_id>(?:[\\d]+|me))/application-passwords": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST", "DELETE"},
			Endpoints: []RouteEndpoint{
				{Methods: []string{"GET"}},
				{Methods: []string{"POST"}},
				{Methods: []string{"DELETE"}},
			},
		},
		"/wp/v2/users/(?P<user_id>(?:[\\d]+|me))/application-passwords/(?P<uuid>[\\w\\-]+)": {
			Namespace: "wp/v2",
			Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Endpoints: []RouteEndpoint{
				{Methods: []string{"GET"}},
				{Methods: []string{"POST", "PUT", "PATCH"}},
				{Methods: []string{"DELETE"}},
			},
		},
	}
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
		"/batch/v1":                                     batchRouteIndex(),
		"/oembed/1.0/proxy":                             getOnly("oembed/1.0"),
		"/wp-site-health/v1":                            getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests":                      getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/background-updates":   getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/loopback-requests":    getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/https-status":         getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/dotorg-communication": getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/authorization-header": getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/directory-sizes":            getOnly("wp-site-health/v1"),
		"/wp-site-health/v1/tests/page-cache":           getOnly("wp-site-health/v1"),
		"/wp-block-editor/v1":                           getOnly("wp-block-editor/v1"),
		"/wp-block-editor/v1/export":                    getOnly("wp-block-editor/v1"),
		"/wp-block-editor/v1/url-details": {
			Namespace: "wp-block-editor/v1",
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{
				Methods: []string{"GET"},
				Args: map[string]any{
					"url": map[string]any{
						"description": "The URL to process.",
						"format":      "uri",
						"required":    true,
						"type":        "string",
					},
				},
			}},
		},
		"/wp-block-editor/v1/navigation-fallback": getOnly("wp-block-editor/v1"),
		"/wp-abilities/v1":                        getOnly("wp-abilities/v1"),
		"/wp-abilities/v1/abilities":              getOnly("wp-abilities/v1"),
		"/wp-abilities/v1/categories":             getOnly("wp-abilities/v1"),
		"/wp/v2/users":                            collectionRoute(ns),
		"/wp/v2/users/(?P<id>[\\d]+)":             itemRoute(ns),
		"/wp/v2/themes":                           getOnly(ns),
		"/wp/v2/themes/(?P<stylesheet>[^\\/:<>\\*\\?\"\\|]+(?:\\/[^\\/:<>\\*\\?\"\\|]+)?)": getOnly(ns),
		"/wp/v2/plugins":                   collectionRoute(ns),
		"/wp/v2/menu-items":                collectionRoute(ns),
		"/wp/v2/menu-items/(?P<id>[\\d]+)": itemRoute(ns),
		"/wp/v2/menus":                     collectionRoute(ns),
		"/wp/v2/menus/(?P<id>[\\d]+)":      itemRoute(ns),
		"/wp/v2/menu-locations": {
			Namespace: ns,
			Methods:   []string{"GET"},
			Endpoints: []RouteEndpoint{{Methods: []string{"GET"}}},
		},
		"/wp/v2/templates": collectionRoute(ns),
		"/wp/v2/templates/(?P<id>([^\\/:<>\\*\\?\"\\|]+(?:\\/[^\\/:<>\\*\\?\"\\|]+)?)[\\/\\w%-]+)": itemRoute(ns),
		"/wp/v2/template-parts": collectionRoute(ns),
		"/wp/v2/template-parts/(?P<id>([^\\/:<>\\*\\?\"\\|]+(?:\\/[^\\/:<>\\*\\?\"\\|]+)?)[\\/\\w%-]+)": itemRoute(ns),
		"/wp/v2/navigation":                collectionRoute(ns),
		"/wp/v2/navigation/(?P<id>[\\d]+)": itemRoute(ns),
		"/wp/v2/global-styles/(?P<id>[\\/\\d+]+)": {
			Namespace: ns,
			Methods:   []string{"GET", "POST", "PUT", "PATCH"},
			Endpoints: []RouteEndpoint{
				{Methods: []string{"GET"}},
				{Methods: []string{"POST", "PUT", "PATCH"}, Args: map[string]any{
					"styles":   map[string]any{"description": "Global styles.", "type": "object", "required": false},
					"settings": map[string]any{"description": "Global settings.", "type": "object", "required": false},
					"title": map[string]any{
						"description": "Title of the global styles variation.",
						"type":        []any{"object", "string"},
						"required":    false,
					},
				}},
			},
		},
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
		// Additional WP route keys (handlers may remain stubs) — P4.33.
		"/wp/v2/block-types/(?P<namespace>[a-zA-Z0-9_-]+)":                          getOnly(ns),
		"/wp/v2/block-types/(?P<namespace>[a-zA-Z0-9_-]+)/(?P<name>[a-zA-Z0-9_-]+)": getOnly(ns),
		"/wp/v2/block-directory/search":                                             getOnly(ns),
		"/wp/v2/pattern-directory/patterns":                                         getOnly(ns),
		"/wp/v2/font-collections":                                                   collectionRoute(ns),
		"/wp/v2/font-collections/(?P<slug>[\\/\\w-]+)":                              itemRoute(ns),
		"/wp/v2/font-families/(?P<font_family_id>[\\d]+)/font-faces":                collectionRoute(ns),
		"/wp/v2/font-families/(?P<font_family_id>[\\d]+)/font-faces/(?P<id>[\\d]+)": itemRoute(ns),
		"/wp/v2/pages/(?P<parent>[\\d]+)/revisions":                                 getOnly(ns),
		"/wp/v2/pages/(?P<parent>[\\d]+)/revisions/(?P<id>[\\d]+)":                  getOnly(ns),
		"/wp/v2/pages/(?P<id>[\\d]+)/autosaves":                                     getOnly(ns),
		"/wp/v2/pages/(?P<parent>[\\d]+)/autosaves/(?P<id>[\\d]+)":                  getOnly(ns),
		"/wp/v2/blocks/(?P<parent>[\\d]+)/revisions":                                getOnly(ns),
		"/wp/v2/blocks/(?P<parent>[\\d]+)/revisions/(?P<id>[\\d]+)":                 getOnly(ns),
		"/wp/v2/blocks/(?P<id>[\\d]+)/autosaves":                                    getOnly(ns),
		"/wp/v2/blocks/(?P<parent>[\\d]+)/autosaves/(?P<id>[\\d]+)":                 getOnly(ns),
		"/wp/v2/media/(?P<id>[\\d]+)/edit":                                          getOnly(ns),
		"/wp/v2/media/(?P<id>[\\d]+)/post-process":                                  getOnly(ns),
		"/wp/v2/menu-locations/(?P<location>[\\w-]+)": {
			Namespace: ns,
			Methods:   []string{"GET", "POST", "PUT", "PATCH"},
			Endpoints: []RouteEndpoint{
				{Methods: []string{"GET"}},
				{Methods: []string{"POST", "PUT", "PATCH"}, Args: map[string]any{
					"menu": map[string]any{
						"description": "The ID of the assigned menu.",
						"type":        "integer",
						"required":    false,
					},
				}},
			},
		},
		"/wp/v2/plugins/(?P<plugin>[^.\\/]+(?:\\/[^.\\/]+)?)":              itemRoute(ns),
		"/wp/v2/global-styles/(?P<parent>[\\d]+)/revisions":                getOnly(ns),
		"/wp/v2/global-styles/(?P<parent>[\\d]+)/revisions/(?P<id>[\\d]+)": getOnly(ns),
		"/wp-abilities/v1/abilities/(?P<name>[a-zA-Z0-9\\-\\/]+)":          getOnly("wp-abilities/v1"),
		"/wp-abilities/v1/abilities/(?P<name>[a-zA-Z0-9\\-\\/]+?)/run":     getOnly("wp-abilities/v1"),
		"/wp-abilities/v1/categories/(?P<slug>[a-z0-9]+(?:-[a-z0-9]+)*)":   getOnly("wp-abilities/v1"),
	}
}

func collectionRoute(ns string) RouteIndex {
	return RouteIndex{
		Namespace: ns,
		Methods:   []string{"GET", "POST"},
		Endpoints: []RouteEndpoint{
			{
				Methods: []string{"GET"},
				Args:    collectionListArgs(),
			},
			{
				Methods: []string{"POST"},
				Args:    collectionCreateArgs(),
			},
		},
	}
}

func itemRoute(ns string) RouteIndex {
	return RouteIndex{
		Namespace: ns,
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []RouteEndpoint{
			{
				Methods: []string{"GET"},
				Args: map[string]any{
					"id": map[string]any{
						"description": "Unique identifier for the post.",
						"type":        "integer",
						"required":    false,
					},
					"context": map[string]any{
						"description": "Scope under which the request is made; determines fields present in response.",
						"type":        "string",
						"default":     "view",
						"required":    false,
						"enum":        []string{"view", "embed", "edit"},
					},
					"password": map[string]any{
						"description": "The password for the post if it is password protected.",
						"type":        "string",
						"required":    false,
					},
				},
			},
			{
				Methods: []string{"POST", "PUT", "PATCH"},
				Args:    collectionCreateArgs(),
			},
			{
				Methods: []string{"DELETE"},
				Args: map[string]any{
					"force": map[string]any{
						"type":        "boolean",
						"default":     false,
						"required":    false,
						"description": "Whether to bypass Trash and force deletion.",
					},
				},
			},
		},
	}
}

// collectionListArgs is the shared WP list query arg catalog (P4.33 expansion).
func collectionListArgs() map[string]any {
	intArray := map[string]any{"type": "array", "default": []any{}, "required": false, "items": map[string]any{"type": "integer"}}
	return map[string]any{
		"context": map[string]any{
			"description": "Scope under which the request is made; determines fields present in response.",
			"type":        "string", "default": "view", "required": false,
			"enum": []string{"view", "embed", "edit"},
		},
		"page":     map[string]any{"description": "Current page of the collection.", "type": "integer", "default": 1, "required": false, "minimum": 1},
		"per_page": map[string]any{"description": "Maximum number of items to be returned in result set.", "type": "integer", "default": 10, "required": false, "minimum": 1, "maximum": 100},
		"search":   map[string]any{"description": "Limit results to those matching a string.", "type": "string", "required": false},
		"after":    map[string]any{"description": "Limit response to posts published after a given ISO8601 compliant date.", "type": "string", "format": "date-time", "required": false},
		"before":   map[string]any{"description": "Limit response to posts published before a given ISO8601 compliant date.", "type": "string", "format": "date-time", "required": false},
		"modified_after": map[string]any{
			"description": "Limit response to posts modified after a given ISO8601 compliant date.",
			"type":        "string", "format": "date-time", "required": false,
		},
		"modified_before": map[string]any{
			"description": "Limit response to posts modified before a given ISO8601 compliant date.",
			"type":        "string", "format": "date-time", "required": false,
		},
		"author":         intArray,
		"author_exclude": intArray,
		"exclude":        intArray,
		"include":        intArray,
		"offset":         map[string]any{"description": "Offset the result set by a specific number of items.", "type": "integer", "required": false},
		"order":          map[string]any{"description": "Order sort attribute ascending or descending.", "type": "string", "default": "desc", "required": false, "enum": []string{"asc", "desc"}},
		"orderby": map[string]any{
			"description": "Sort collection by post attribute.",
			"type":        "string", "default": "date", "required": false,
			"enum": []string{"author", "date", "id", "include", "modified", "parent", "relevance", "slug", "include_slugs", "title"},
		},
		"search_columns":     map[string]any{"type": "array", "default": []any{}, "required": false, "items": map[string]any{"type": "string"}},
		"search_semantics":   map[string]any{"description": "How to interpret the search input.", "type": "string", "required": false, "enum": []string{"exact"}},
		"slug":               map[string]any{"description": "Limit result set to posts with one or more specific slugs.", "type": "array", "required": false, "items": map[string]any{"type": "string"}},
		"status":             map[string]any{"description": "Limit result set to posts assigned one or more statuses.", "type": "array", "default": "publish", "required": false},
		"tax_relation":       map[string]any{"description": "Limit result set based on relationship between multiple taxonomies.", "type": "string", "required": false, "enum": []string{"AND", "OR"}},
		"categories":         intArray,
		"categories_exclude": intArray,
		"tags":               intArray,
		"tags_exclude":       intArray,
		"sticky":             map[string]any{"description": "Limit result set to items that are sticky.", "type": "boolean", "required": false},
		"ignore_sticky":      map[string]any{"description": "Whether to ignore sticky posts or not.", "type": "boolean", "default": false, "required": false},
		"format":             map[string]any{"description": "Limit result set to items assigned one or more given formats.", "type": "array", "required": false, "items": map[string]any{"type": "string"}},
		"parent":             intArray,
		"parent_exclude":     intArray,
		"menu_order":         map[string]any{"type": "integer", "required": false},
		"media_type":         map[string]any{"type": "string", "required": false},
		"mime_type":          map[string]any{"type": "string", "required": false},
		"post":               intArray,
		"type":               map[string]any{"type": "string", "required": false},
		"author_email":       map[string]any{"type": "string", "format": "email", "required": false},
		"password":           map[string]any{"type": "string", "required": false},
	}
}

func collectionCreateArgs() map[string]any {
	return map[string]any{
		"date":           map[string]any{"description": "The date the post was published, in the site's timezone.", "type": []any{"string", "null"}, "format": "date-time", "required": false},
		"date_gmt":       map[string]any{"description": "The date the post was published, as GMT.", "type": []any{"string", "null"}, "format": "date-time", "required": false},
		"slug":           map[string]any{"description": "An alphanumeric identifier for the post unique to its type.", "type": "string", "required": false},
		"status":         map[string]any{"description": "A named status for the post.", "type": "string", "required": false, "enum": []string{"publish", "future", "draft", "pending", "private"}},
		"password":       map[string]any{"description": "A password to protect access to the content and excerpt.", "type": "string", "required": false},
		"title":          map[string]any{"description": "The title for the post.", "type": "object", "required": false},
		"content":        map[string]any{"description": "The content for the post.", "type": "object", "required": false},
		"author":         map[string]any{"description": "The ID for the author of the post.", "type": "integer", "required": false},
		"excerpt":        map[string]any{"description": "The excerpt for the post.", "type": "object", "required": false},
		"featured_media": map[string]any{"description": "The ID of the featured media for the post.", "type": "integer", "required": false},
		"comment_status": map[string]any{"description": "Whether or not comments are open on the post.", "type": "string", "required": false, "enum": []string{"open", "closed"}},
		"ping_status":    map[string]any{"description": "Whether or not the post can be pinged.", "type": "string", "required": false, "enum": []string{"open", "closed"}},
		"format":         map[string]any{"description": "The format for the post.", "type": "string", "required": false},
		"meta":           map[string]any{"description": "Meta fields.", "type": "object", "required": false},
		"sticky":         map[string]any{"description": "Whether or not the post should be treated as sticky.", "type": "boolean", "required": false},
		"template":       map[string]any{"description": "The theme file to use to display the post.", "type": "string", "required": false},
		"categories":     map[string]any{"description": "The terms assigned to the post in the category taxonomy.", "type": "array", "required": false, "items": map[string]any{"type": "integer"}},
		"tags":           map[string]any{"description": "The terms assigned to the post in the post_tag taxonomy.", "type": "array", "required": false, "items": map[string]any{"type": "integer"}},
		"parent":         map[string]any{"type": "integer", "required": false},
		"menu_order":     map[string]any{"type": "integer", "required": false},
		"alt_text":       map[string]any{"type": "string", "required": false},
		"caption":        map[string]any{"type": "object", "required": false},
		"description":    map[string]any{"type": "object", "required": false},
		"post":           map[string]any{"type": "integer", "required": false},
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
		"/wp/v2/themes/(?P<stylesheet>[^\\/:<>\\*\\?\"\\|]+(?:\\/[^\\/:<>\\*\\?\"\\|]+)?)": {
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
