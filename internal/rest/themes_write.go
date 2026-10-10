package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// updateTheme handles POST/PUT/PATCH on /wp/v2/themes/{stylesheet} (P4.78).
// Supports status: "active" (switch_theme). Inactive is rejected — a theme is
// deactivated only by activating another.
func (s *Server) updateTheme(w http.ResponseWriter, r *http.Request, c *wp.Context, stylesheet string) {
	if !canSwitchThemes(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_manage_themes",
			"message": "Sorry, you are not allowed to manage themes for this site.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	stylesheet, _ = url.PathUnescape(strings.TrimSpace(stylesheet))
	stylesheet = strings.Trim(stylesheet, "/")
	if stylesheet == "" || strings.Contains(stylesheet, "..") || strings.ContainsAny(stylesheet, `/\`) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_theme_not_found",
			"message": "Theme not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	abs := wp.AbsPath(c)
	stylePath := filepath.Join(theme.ThemesDir(abs), stylesheet, "style.css")
	if st, err := os.Stat(stylePath); err != nil || st.IsDir() {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_theme_not_found",
			"message": "Theme not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}

	body, err := decodeThemeBody(r)
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
		switch status {
		case "active":
			if wp.ActiveStylesheet(c) != stylesheet {
				if err := wp.SwitchTheme(c, abs, stylesheet); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"code":    "rest_cannot_activate_theme",
						"message": err.Error(),
						"data":    map[string]any{"status": 500},
					})
					return
				}
			}
		case "inactive":
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_theme_invalid_status",
				"message": "Themes cannot be set inactive; activate another theme instead.",
				"data":    map[string]any{"status": 400},
			})
			return
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "rest_invalid_param",
				"message": fmt.Sprintf("Invalid status %q; expected active.", status),
				"data":    map[string]any{"status": 400},
			})
			return
		}
	}

	data, err := theme.GetThemeData(stylePath)
	if err != nil || data.Name == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_theme_not_found",
			"message": "Theme not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	data.Slug = stylesheet
	active := wp.ActiveStylesheet(c)
	w.Header().Set("Allow", "GET, POST, PUT, PATCH")
	writeJSON(w, http.StatusOK, prepareTheme(c, abs, data, active))
}

func decodeThemeBody(r *http.Request) (map[string]any, error) {
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
