package rest

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// PluginREST is a wp/v2 plugins object (P4.60).
type PluginREST struct {
	Plugin      string            `json:"plugin"`
	Status      string            `json:"status"`
	Name        string            `json:"name"`
	PluginURI   string            `json:"plugin_uri"`
	Author      string            `json:"author"`
	AuthorURI   string            `json:"author_uri"`
	Description PluginDescription `json:"description"`
	Version     string            `json:"version"`
	NetworkOnly bool              `json:"network_only"`
	RequiresWP  string            `json:"requires_wp"`
	RequiresPHP string            `json:"requires_php"`
	TextDomain  string            `json:"textdomain"`
	Links       map[string]any    `json:"_links"`
}

// PluginDescription is raw/rendered plugin description.
type PluginDescription struct {
	Raw      string `json:"raw"`
	Rendered string `json:"rendered"`
}

func (s *Server) servePlugins(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canActivatePlugins(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view_plugins",
			"message": "Sorry, you are not allowed to manage plugins for this site.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	installed, err := plugins.ListInstalledPlugins(c)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_plugin_error",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	statusFilter := pluginStatusFilter(r)
	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))
	out := make([]PluginREST, 0, len(installed))
	for _, p := range installed {
		item := preparePlugin(c, p)
		if len(statusFilter) > 0 && !containsString(statusFilter, item.Status) {
			continue
		}
		if search != "" {
			hay := strings.ToLower(item.Name + " " + item.Description.Raw + " " + item.Author + " " + item.Plugin)
			if !strings.Contains(hay, search) {
				continue
			}
		}
		out = append(out, item)
	}
	w.Header().Set("X-WP-Total", strconv.Itoa(len(out)))
	w.Header().Set("X-WP-TotalPages", "1")
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) servePluginItem(w http.ResponseWriter, r *http.Request, c *wp.Context, plugin string) {
	switch r.Method {
	case http.MethodGet:
		// continue
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		s.updatePlugin(w, r, c, plugin)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canActivatePlugins(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view_plugin",
			"message": "Sorry, you are not allowed to manage plugins for this site.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	p, _, err := findInstalledPlugin(c, plugin)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_plugin_error",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if p == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_plugin_not_found",
			"message": "Plugin not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, preparePlugin(c, *p))
}

func preparePlugin(c *wp.Context, p plugins.PluginData) PluginREST {
	base := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	file := p.File
	pluginKey := strings.TrimSuffix(file, ".php")
	status := "inactive"
	if plugins.IsPluginActive(c, file) {
		status = "active"
	}
	desc := p.Description
	return PluginREST{
		Plugin:      pluginKey,
		Status:      status,
		Name:        p.Name,
		PluginURI:   p.PluginURI,
		Author:      p.Author,
		AuthorURI:   p.AuthorURI,
		Description: PluginDescription{Raw: desc, Rendered: desc},
		Version:     p.Version,
		NetworkOnly: p.Network,
		RequiresWP:  p.RequiresWP,
		RequiresPHP: p.RequiresPHP,
		TextDomain:  p.TextDomain,
		Links: map[string]any{
			"self": []map[string]any{{
				"href": base + "/wp-json/wp/v2/plugins/" + url.PathEscape(pluginKey),
			}},
			"collection": []map[string]any{{
				"href": base + "/wp-json/wp/v2/plugins",
			}},
		},
	}
}

func canActivatePlugins(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, "activate_plugins")
}

func pluginStatusFilter(r *http.Request) []string {
	q := r.URL.Query()
	vals := q["status"]
	if len(vals) == 0 {
		vals = q["status[]"]
	}
	var out []string
	for _, v := range vals {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(strings.ToLower(part))
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}
