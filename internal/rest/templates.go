package rest

import (
	"net/http"
	"sort"
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// BlockTemplate is a wp/v2 templates or template-parts object (P4.23 read stub).
type BlockTemplate struct {
	ID           string            `json:"id"`
	Theme        string            `json:"theme"`
	Slug         string            `json:"slug"`
	Source       string            `json:"source"`
	Type         string            `json:"type"`
	Title        map[string]string `json:"title"`
	Content      map[string]any    `json:"content"`
	Status       string            `json:"status"`
	WPID         int64             `json:"wp_id"`
	HasThemeFile bool              `json:"has_theme_file"`
	IsCustom     bool              `json:"is_custom,omitempty"`
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
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewTemplates(c) {
		writeTemplatesForbidden(w, c)
		return
	}
	out := listBlockTemplates(c, postType)
	w.Header().Set("Allow", "GET")
	setCollectionHeaders(w, r, c, len(out), 1, 1)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveTemplateItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	s.serveBlockTemplateItem(w, r, c, "wp_template", id)
}

func (s *Server) serveTemplatePartItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id string) {
	s.serveBlockTemplateItem(w, r, c, "wp_template_part", id)
}

func (s *Server) serveBlockTemplateItem(w http.ResponseWriter, r *http.Request, c *wp.Context, postType, id string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewTemplates(c) {
		writeTemplatesForbidden(w, c)
		return
	}
	id = strings.Trim(id, "/")
	for _, t := range listBlockTemplates(c, postType) {
		if t.ID == id || t.Slug == id {
			w.Header().Set("Allow", "GET")
			writeJSON(w, http.StatusOK, t)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]any{
		"code":    "rest_template_not_found",
		"message": "No templates exist with that id.",
		"data":    map[string]any{"status": 404},
	})
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
			Type:         postType,
			Title:        map[string]string{"raw": title, "rendered": title},
			Content:      map[string]any{"raw": f.Content, "block_version": 1},
			Status:       "publish",
			WPID:         0,
			HasThemeFile: true,
			IsCustom:     false,
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
			title := row.Title
			if title == "" {
				title = humanizeSlug(row.Name)
			}
			_, hasFile := bySlug[row.Name]
			bt := BlockTemplate{
				ID:           stylesheet + "//" + row.Name,
				Theme:        stylesheet,
				Slug:         row.Name,
				Source:       "custom",
				Type:         postType,
				Title:        map[string]string{"raw": title, "rendered": title},
				Content:      map[string]any{"raw": row.Content, "block_version": 1},
				Status:       "publish",
				WPID:         row.ID,
				HasThemeFile: hasFile,
				IsCustom:     postType == "wp_template" && !hasFile,
			}
			if postType == "wp_template_part" {
				bt.Area = inferTemplatePartArea(row.Name)
			}
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
