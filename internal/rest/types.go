package rest

import (
	"net/http"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PostType is a minimal wp/v2 type object (view context).
type PostType struct {
	Description   string   `json:"description"`
	Hierarchical  bool     `json:"hierarchical"`
	HasArchive    any      `json:"has_archive"`
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	Icon          string   `json:"icon"`
	Taxonomies    []string `json:"taxonomies"`
	RestBase      string   `json:"rest_base"`
	RestNamespace string   `json:"rest_namespace"`
	Template      []any    `json:"template"`
	TemplateLock  any      `json:"template_lock"`
}

func (s *Server) serveTypes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	writeJSON(w, http.StatusOK, builtinPostTypes())
}

func builtinPostTypes() map[string]PostType {
	return map[string]PostType{
		"post": {
			Description:   "",
			Hierarchical:  false,
			HasArchive:    false,
			Name:          "Posts",
			Slug:          "post",
			Icon:          "dashicons-admin-post",
			Taxonomies:    []string{"category", "post_tag"},
			RestBase:      "posts",
			RestNamespace: "wp/v2",
			Template:      []any{},
			TemplateLock:  false,
		},
		"page": {
			Description:   "",
			Hierarchical:  true,
			HasArchive:    false,
			Name:          "Pages",
			Slug:          "page",
			Icon:          "dashicons-admin-page",
			Taxonomies:    []string{},
			RestBase:      "pages",
			RestNamespace: "wp/v2",
			Template:      []any{},
			TemplateLock:  false,
		},
		"attachment": {
			Description:   "",
			Hierarchical:  false,
			HasArchive:    false,
			Name:          "Media",
			Slug:          "attachment",
			Icon:          "dashicons-admin-media",
			Taxonomies:    []string{},
			RestBase:      "media",
			RestNamespace: "wp/v2",
			Template:      []any{},
			TemplateLock:  false,
		},
		"wp_block": {
			Description:   "",
			Hierarchical:  false,
			HasArchive:    false,
			Name:          "Patterns",
			Slug:          "wp_block",
			Icon:          "dashicons-layout",
			Taxonomies:    []string{"wp_pattern_category"},
			RestBase:      "blocks",
			RestNamespace: "wp/v2",
			Template:      []any{},
			TemplateLock:  false,
		},
	}
}
