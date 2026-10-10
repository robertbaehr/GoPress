package rest

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// ThemeREST is a wp/v2 themes object (P4.59).
type ThemeREST struct {
	Stylesheet    string         `json:"stylesheet"`
	Template      string         `json:"template"`
	Version       string         `json:"version"`
	RequiresPHP   string         `json:"requires_php"`
	RequiresWP    string         `json:"requires_wp"`
	TextDomain    string         `json:"textdomain"`
	Screenshot    string         `json:"screenshot"`
	Author        ThemeRichField `json:"author"`
	AuthorURI     ThemeRichField `json:"author_uri"`
	Description   ThemeRichField `json:"description"`
	Name          ThemeRichField `json:"name"`
	Tags          ThemeRichField `json:"tags"`
	ThemeURI      ThemeRichField `json:"theme_uri"`
	Status        string         `json:"status"`
	IsBlockTheme  bool           `json:"is_block_theme"`
	StylesheetURI string         `json:"stylesheet_uri"`
	TemplateURI   string         `json:"template_uri"`
	ThemeSupports map[string]any `json:"theme_supports,omitempty"`
	Links         map[string]any `json:"_links"`
}

// ThemeRichField is a raw/rendered theme header pair.
type ThemeRichField struct {
	Raw      string `json:"raw"`
	Rendered string `json:"rendered"`
}

func (s *Server) serveThemes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	statusFilter := themeStatusFilter(r)
	canSwitch := canSwitchThemes(c)
	canReadActive := canEditPosts(c)
	if !canSwitch {
		if !(len(statusFilter) == 1 && statusFilter[0] == "active" && canReadActive) {
			status := http.StatusUnauthorized
			if c != nil && c.CurrentUser != nil {
				status = http.StatusForbidden
			}
			code := "rest_cannot_view_themes"
			msg := "Sorry, you are not allowed to view themes."
			if len(statusFilter) == 1 && statusFilter[0] == "active" {
				code = "rest_cannot_view_active_theme"
				msg = "Sorry, you are not allowed to view the active theme."
			}
			writeJSON(w, status, map[string]any{
				"code":    code,
				"message": msg,
				"data":    map[string]any{"status": status},
			})
			return
		}
	}

	abs := wp.AbsPath(c)
	active := wp.ActiveStylesheet(c)
	installed, err := theme.ListInstalledThemes(abs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_theme_error",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}

	out := make([]ThemeREST, 0, len(installed))
	if !canSwitch {
		// Active-only listing for editors without switch_themes.
		for _, t := range installed {
			if t.Slug == active {
				out = append(out, prepareTheme(c, abs, t, active))
				break
			}
		}
	} else {
		for _, t := range installed {
			st := "inactive"
			if t.Slug == active {
				st = "active"
			}
			if len(statusFilter) > 0 && !containsString(statusFilter, st) {
				continue
			}
			out = append(out, prepareTheme(c, abs, t, active))
		}
	}
	w.Header().Set("X-WP-Total", strconv.Itoa(len(out)))
	w.Header().Set("X-WP-TotalPages", "1")
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveThemeItem(w http.ResponseWriter, r *http.Request, c *wp.Context, stylesheet string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	stylesheet, _ = url.PathUnescape(strings.TrimSpace(stylesheet))
	stylesheet = strings.Trim(stylesheet, "/")
	if stylesheet == "" || strings.Contains(stylesheet, "..") {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_theme_not_found",
			"message": "Theme not found.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	abs := wp.AbsPath(c)
	active := wp.ActiveStylesheet(c)
	canSwitch := canSwitchThemes(c)
	if !canSwitch && stylesheet != active {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view_themes",
			"message": "Sorry, you are not allowed to view themes.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	if !canSwitch && !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view_active_theme",
			"message": "Sorry, you are not allowed to view the active theme.",
			"data":    map[string]any{"status": status},
		})
		return
	}

	stylePath := filepath.Join(theme.ThemesDir(abs), stylesheet, "style.css")
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
	writeJSON(w, http.StatusOK, prepareTheme(c, abs, data, active))
}

func prepareTheme(c *wp.Context, abs string, t theme.ThemeData, active string) ThemeREST {
	status := "inactive"
	if t.Slug == active {
		status = "active"
	}
	tmpl := strings.TrimSpace(t.Template)
	if tmpl == "" {
		tmpl = t.Slug
	}
	base := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	ssURI := base + "/wp-content/themes/" + t.Slug
	tmplURI := base + "/wp-content/themes/" + tmpl
	rich := func(s string) ThemeRichField {
		s = strings.TrimSpace(s)
		return ThemeRichField{Raw: s, Rendered: s}
	}
	out := ThemeREST{
		Stylesheet:    t.Slug,
		Template:      tmpl,
		Version:       t.Version,
		RequiresPHP:   t.RequiresPHP,
		RequiresWP:    t.RequiresWP,
		TextDomain:    t.TextDomain,
		Screenshot:    themeScreenshotURL(base, abs, t.Slug),
		Author:        rich(t.Author),
		AuthorURI:     rich(t.AuthorURI),
		Description:   rich(t.Description),
		Name:          rich(t.Name),
		Tags:          rich(t.Tags),
		ThemeURI:      rich(t.ThemeURI),
		Status:        status,
		IsBlockTheme:  theme.IsBlockTheme(abs, t.Slug),
		StylesheetURI: ssURI,
		TemplateURI:   tmplURI,
		Links: map[string]any{
			"self": []map[string]any{{
				"href": base + "/wp-json/wp/v2/themes/" + url.PathEscape(t.Slug),
			}},
			"collection": []map[string]any{{
				"href": base + "/wp-json/wp/v2/themes",
			}},
		},
	}
	if status == "active" {
		out.ThemeSupports = map[string]any{}
	}
	return out
}

func themeScreenshotURL(base, abs, slug string) string {
	dir := filepath.Join(theme.ThemesDir(abs), slug)
	for _, name := range []string{"screenshot.png", "screenshot.jpg", "screenshot.jpeg", "screenshot.gif", "screenshot.webp"} {
		if st, err := os.Stat(filepath.Join(dir, name)); err == nil && !st.IsDir() {
			return base + "/wp-content/themes/" + slug + "/" + name
		}
	}
	return ""
}

func canSwitchThemes(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, "switch_themes")
}

func themeStatusFilter(r *http.Request) []string {
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

func containsString(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
