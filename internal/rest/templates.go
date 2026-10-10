package rest

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// BlockTemplate is a wp/v2 templates or template-parts object (P4.23 / P4.112).
// P4.112: description, origin, author, modified, date (WP_REST_Templates_Controller).
type BlockTemplate struct {
	ID           string            `json:"id"`
	Theme        string            `json:"theme"`
	Slug         string            `json:"slug"`
	Source       string            `json:"source"`
	Origin       *string           `json:"origin"`
	Type         string            `json:"type"`
	Description  string            `json:"description"`
	Title        map[string]string `json:"title"`
	Content      map[string]any    `json:"content"`
	Status       string            `json:"status"`
	WPID         int64             `json:"wp_id"`
	HasThemeFile bool              `json:"has_theme_file"`
	IsCustom     bool              `json:"is_custom,omitempty"`
	Author       int64             `json:"author"`
	Modified     *string           `json:"modified"`
	Date         *string           `json:"date"`
	Area         string            `json:"area,omitempty"`
}

func canViewTemplates(c *wp.Context) bool {
	return canViewMenus(c)
}

func writeTemplatesForbidden(w http.ResponseWriter, c *wp.Context) {
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]any{
		"code":    "rest_cannot_manage_templates",
		"message": "Sorry, you are not allowed to access the templates on this site.",
		"data":    map[string]any{"status": status},
	})
}

func (s *Server) serveTemplates(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveBlockTemplateCollection(w, r, c, "wp_template")
}

func (s *Server) serveTemplateParts(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.serveBlockTemplateCollection(w, r, c, "wp_template_part")
}

func (s *Server) serveBlockTemplateCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !canViewTemplates(c) {
			writeTemplatesForbidden(w, c)
			return
		}
		out := listBlockTemplates(c, postType)
		// P4.112: collection filters area / wp_id (get_collection_params).
		out = filterBlockTemplates(out, r)
		w.Header().Set("Allow", "GET, POST")
		setCollectionHeaders(w, r, c, len(out), 1, 1)
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		s.createBlockTemplate(w, r, c, postType)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

// filterBlockTemplates applies REST collection query args area and wp_id (P4.112).
func filterBlockTemplates(in []BlockTemplate, r *http.Request) []BlockTemplate {
	if r == nil {
		return in
	}
	q := r.URL.Query()
	area := strings.TrimSpace(q.Get("area"))
	wpIDRaw := strings.TrimSpace(q.Get("wp_id"))
	if area == "" && wpIDRaw == "" {
		return in
	}
	var wantWPID int64
	filterWPID := false
	if wpIDRaw != "" {
		filterWPID = true
		n, err := strconv.ParseInt(wpIDRaw, 10, 64)
		if err != nil {
			wantWPID = -1
		} else {
			wantWPID = n
		}
	}
	out := make([]BlockTemplate, 0, len(in))
	for _, t := range in {
		if area != "" && t.Area != area {
			continue
		}
		if filterWPID && t.WPID != wantWPID {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (s *Server) serveTemplateItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	s.serveBlockTemplateItem(w, r, c, "wp_template", id)
}

func (s *Server) serveTemplatePartItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	s.serveBlockTemplateItem(w, r, c, "wp_template_part", id)
}

func (s *Server) serveBlockTemplateItem(w http.ResponseWriter, r *http.Request, c *wp.Context, postType, id string) {
	switch r.Method {
	case http.MethodGet:
		if !canViewTemplates(c) {
			writeTemplatesForbidden(w, c)
			return
		}
		id = strings.Trim(id, "/")
		for _, t := range listBlockTemplates(c, postType) {
			if t.ID == id || t.Slug == id {
				w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
				writeJSON(w, http.StatusOK, t)
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_template_not_found",
			"message": "No templates exist with that id.",
			"data":    map[string]any{"status": 404},
		})
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		s.updateBlockTemplate(w, r, c, postType, id)
	case http.MethodDelete:
		s.deleteBlockTemplate(w, r, c, postType, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

func listBlockTemplates(c *wp.Context, postType string) []BlockTemplate {
	stylesheet := wp.ActiveStylesheet(c)
	abs := wp.AbsPath(c)
	bySlug := map[string]BlockTemplate{}

	var files []theme.HTMLFile
	if postType == "wp_template_part" {
		files = theme.ListPartHTML(abs, stylesheet)
	} else {
		files = theme.ListTemplateHTML(abs, stylesheet)
	}
	for _, f := range files {
		title := humanizeSlug(f.Slug)
		bt := BlockTemplate{
			ID:           stylesheet + "//" + f.Slug,
			Theme:        stylesheet,
			Slug:         f.Slug,
			Source:       "theme",
			Origin:       nil,
			Type:         postType,
			Description:  "",
			Title:        map[string]string{"raw": title, "rendered": title},
			Content:      map[string]any{"raw": f.Content, "block_version": 1},
			Status:       "publish",
			WPID:         0,
			HasThemeFile: true,
			IsCustom:     false,
			Author:       0,
			Modified:     nil,
			Date:         nil,
		}
		if postType == "wp_template_part" {
			bt.Area = inferTemplatePartArea(f.Slug)
		}
		bySlug[f.Slug] = bt
	}

	rows, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     postType,
		PostStatus:   "publish",
		PostsPerPage: 100,
	})
	if err == nil {
		for _, row := range rows {
			_, hasFile := bySlug[row.Name]
			bt := blockTemplateFromPost(stylesheet, postType, row, hasFile)
			bySlug[row.Name] = bt
		}
	}

	out := make([]BlockTemplate, 0, len(bySlug))
	for _, v := range bySlug {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

func inferTemplatePartArea(slug string) string {
	slug = strings.ToLower(slug)
	switch {
	case strings.Contains(slug, "header"):
		return "header"
	case strings.Contains(slug, "footer"):
		return "footer"
	case strings.Contains(slug, "sidebar"):
		return "uncategorized"
	default:
		return "uncategorized"
	}
}

// blockTemplateFromPost builds a custom (DB-backed) BlockTemplate (P4.112 fields).
func blockTemplateFromPost(stylesheet, postType string, row wp.Post, hasFile bool) BlockTemplate {
	title := row.Title
	if title == "" {
		title = humanizeSlug(row.Name)
	}
	bt := BlockTemplate{
		ID:           stylesheet + "//" + row.Name,
		Theme:        stylesheet,
		Slug:         row.Name,
		Source:       "custom",
		Type:         postType,
		Description:  strings.TrimSpace(row.Excerpt),
		Title:        map[string]string{"raw": title, "rendered": title},
		Content:      map[string]any{"raw": row.Content, "block_version": 1},
		Status:       row.Status,
		WPID:         row.ID,
		HasThemeFile: hasFile,
		IsCustom:     postType == "wp_template" && !hasFile,
		Author:       row.Author,
	}
	if bt.Status == "" {
		bt.Status = "publish"
	}
	// Customized theme file keeps origin=theme (WP prepare_item_for_response).
	if hasFile {
		origin := "theme"
		bt.Origin = &origin
	}
	if mod := mysqlDateToREST(row.ModifiedGMT); mod != "" {
		bt.Modified = &mod
	} else if mod := mysqlDateToREST(row.Modified); mod != "" {
		bt.Modified = &mod
	}
	if date := mysqlDateToREST(row.DateGMT); date != "" {
		bt.Date = &date
	} else if date := mysqlDateToREST(row.Date); date != "" {
		bt.Date = &date
	}
	if postType == "wp_template_part" {
		bt.Area = inferTemplatePartArea(row.Name)
	}
	return bt
}

func humanizeSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func parseTemplateIDPath(path, prefix string) (id string, ok bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" || strings.Contains(rest, "/autosaves") || strings.Contains(rest, "/revisions") {
		return "", false
	}
	return rest, true
}
