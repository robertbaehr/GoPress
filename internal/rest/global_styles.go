package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// GlobalStylesTheme is GET /wp/v2/global-styles/themes/{stylesheet} (P4.22).
type GlobalStylesTheme struct {
	Settings map[string]any `json:"settings"`
	Styles   map[string]any `json:"styles"`
}

// GlobalStylesItem is GET /wp/v2/global-styles/{id} for a wp_global_styles post.
type GlobalStylesItem struct {
	ID       int64             `json:"id"`
	Title    map[string]string `json:"title"`
	Settings map[string]any    `json:"settings"`
	Styles   map[string]any    `json:"styles"`
}

func canViewGlobalStyles(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	if c.UserCan(c.CurrentUser, "edit_theme_options") {
		return true
	}
	return c.UserCan(c.CurrentUser, "edit_posts")
}

func writeGlobalStylesForbidden(w http.ResponseWriter, c *wp.Context, code, message string) {
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
		"data":    map[string]any{"status": status},
	})
}

func (s *Server) serveGlobalStylesTheme(w http.ResponseWriter, r *http.Request, c *wp.Context, stylesheet string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewGlobalStyles(c) {
		writeGlobalStylesForbidden(w, c, "rest_cannot_read_global_styles",
			"Sorry, you are not allowed to access the global styles on this site.")
		return
	}
	stylesheet = strings.TrimSpace(stylesheet)
	if stylesheet == "" || strings.Contains(stylesheet, "..") || strings.ContainsAny(stylesheet, `/\`) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_theme_not_found",
			"message": "Theme not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	abs := ""
	if c != nil && c.Config != nil {
		abs = c.Config.ABSPATH
	}
	tj, err := theme.LoadActive(abs, stylesheet)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_theme_not_found",
			"message": "Theme not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	settings := tj.Settings
	if settings == nil {
		settings = map[string]any{}
	}
	styles := tj.Styles
	if styles == nil {
		styles = map[string]any{}
	}
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, GlobalStylesTheme{Settings: settings, Styles: styles})
}

func (s *Server) serveGlobalStylesItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewGlobalStyles(c) {
		writeGlobalStylesForbidden(w, c, "rest_cannot_view",
			"Sorry, you are not allowed to view this global style.")
		return
	}
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "wp_global_styles", PostStatus: "publish"})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if row == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	settings, styles := parseGlobalStylesContent(row.Content)
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, GlobalStylesItem{
		ID:       row.ID,
		Title:    map[string]string{"raw": row.Title, "rendered": row.Title},
		Settings: settings,
		Styles:   styles,
	})
}

func parseGlobalStylesContent(content string) (settings, styles map[string]any) {
	settings = map[string]any{}
	styles = map[string]any{}
	content = strings.TrimSpace(content)
	if content == "" {
		return settings, styles
	}
	var shape struct {
		Settings map[string]any `json:"settings"`
		Styles   map[string]any `json:"styles"`
	}
	if err := json.Unmarshal([]byte(content), &shape); err != nil {
		return settings, styles
	}
	if shape.Settings != nil {
		settings = shape.Settings
	}
	if shape.Styles != nil {
		styles = shape.Styles
	}
	return settings, styles
}

// parseGlobalStylesPath matches themes/{stylesheet} or numeric id under /wp/v2/global-styles/.
func parseGlobalStylesPath(path string) (kind string, stylesheet string, id int64, ok bool) {
	rest := strings.TrimPrefix(path, "/wp/v2/global-styles/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", "", 0, false
	}
	if strings.HasPrefix(rest, "themes/") {
		ss := strings.TrimPrefix(rest, "themes/")
		ss = strings.Trim(ss, "/")
		if ss == "" || strings.Contains(ss, "/") {
			return "", "", 0, false
		}
		return "theme", ss, 0, true
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		return "", "", 0, false
	}
	return "id", "", id, true
}
