package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// updatePlugin handles PUT/PATCH/POST on /wp/v2/plugins/{plugin} (P4.77).
// Supports status: "active" | "inactive" (activate/deactivate).
func (s *Server) updatePlugin(w http.ResponseWriter, r *http.Request, c *wp.Context, plugin string) {
	if !canActivatePlugins(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_manage_plugins",
			"message": "Sorry, you are not allowed to manage plugins for this site.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	p, file, err := findInstalledPlugin(c, plugin)
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

	body, err := decodePluginBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_json",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	status, hasStatus := body["status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))
	if hasStatus && status != "" {
		current := "inactive"
		if plugins.IsPluginActive(c, file) {
			current = "active"
		}
		if status != current {
			switch status {
			case "active":
				if err := plugins.ActivatePlugin(c, file); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"code":    "rest_cannot_activate_plugin",
						"message": err.Error(),
						"data":    map[string]any{"status": 500},
					})
					return
				}
			case "inactive":
				if err := plugins.DeactivatePlugin(c, file); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"code":    "rest_cannot_deactivate_plugin",
						"message": err.Error(),
						"data":    map[string]any{"status": 500},
					})
					return
				}
			default:
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"code":    "rest_invalid_param",
					"message": fmt.Sprintf("Invalid status %q; expected active or inactive.", status),
					"data":    map[string]any{"status": 400},
				})
				return
			}
		}
	}
	// Reload after status change.
	p2, _, err := findInstalledPlugin(c, plugin)
	if err != nil || p2 == nil {
		p2 = p
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, preparePlugin(c, *p2))
}

func findInstalledPlugin(c *wp.Context, plugin string) (*plugins.PluginData, string, error) {
	plugin, _ = url.PathUnescape(strings.TrimSpace(plugin))
	plugin = strings.Trim(plugin, "/")
	if plugin == "" || strings.Contains(plugin, "..") {
		return nil, "", nil
	}
	file := plugin
	if !strings.HasSuffix(strings.ToLower(file), ".php") {
		file += ".php"
	}
	installed, err := plugins.ListInstalledPlugins(c)
	if err != nil {
		return nil, "", err
	}
	for i := range installed {
		p := &installed[i]
		if p.File == file || strings.TrimSuffix(p.File, ".php") == plugin {
			return p, p.File, nil
		}
	}
	return nil, file, nil
}

func decodePluginBody(r *http.Request) (map[string]any, error) {
	out := map[string]any{}
	if r == nil || r.Body == nil {
		return out, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON body")
	}
	return out, nil
}
