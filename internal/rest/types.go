package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PostType is a wp/v2 type object (view context).
type PostType struct {
	Description   string         `json:"description"`
	Hierarchical  bool           `json:"hierarchical"`
	HasArchive    any            `json:"has_archive"`
	Name          string         `json:"name"`
	Slug          string         `json:"slug"`
	Icon          any            `json:"icon"`
	Taxonomies    []string       `json:"taxonomies"`
	RestBase      string         `json:"rest_base"`
	RestNamespace string         `json:"rest_namespace"`
	Template      []any          `json:"template"`
	TemplateLock  any            `json:"template_lock"`
	Links         map[string]any `json:"_links"`
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
	writeJSON(w, http.StatusOK, builtinPostTypes(c))
}

func builtinPostTypes(c *wp.Context) map[string]PostType {
	base := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	restRoot := base + "/wp-json"
	typesBase := restRoot + "/wp/v2/types"

	mk := func(description, name, slug string, icon any, hierarchical bool, taxonomies []string, restBase string) PostType {
		if taxonomies == nil {
			taxonomies = []string{}
		}
		return PostType{
			Description:   description,
			Hierarchical:  hierarchical,
			HasArchive:    false,
			Name:          name,
			Slug:          slug,
			Icon:          icon,
			Taxonomies:    taxonomies,
			RestBase:      restBase,
			RestNamespace: "wp/v2",
			Template:      []any{},
			TemplateLock:  false,
			Links: map[string]any{
				"collection": []map[string]any{{"href": typesBase}},
				"wp:items":   []map[string]any{{"href": restRoot + "/wp/v2/" + restBase}},
				"curies": []map[string]any{{
					"name":      "wp",
					"href":      "https://api.w.org/{rel}",
					"templated": true,
				}},
			},
		}
	}

	return map[string]PostType{
		"post": mk("", "Posts", "post", "dashicons-admin-post", false, []string{"category", "post_tag"}, "posts"),
		"page": mk("", "Pages", "page", "dashicons-admin-page", true, []string{}, "pages"),
		"attachment": mk("", "Media", "attachment", "dashicons-admin-media", false, []string{}, "media"),
		"nav_menu_item": mk("", "Navigation Menu Items", "nav_menu_item", nil, false, []string{"nav_menu"}, "menu-items"),
		"wp_block": mk("", "Patterns", "wp_block", nil, false, []string{"wp_pattern_category"}, "blocks"),
		"wp_template": mk("Templates to include in your theme.", "Templates", "wp_template", nil, false, []string{}, "templates"),
		"wp_template_part": mk("Template parts to include in your templates.", "Template Parts", "wp_template_part", nil, false, []string{}, "template-parts"),
		"wp_global_styles": mk("Global styles to include in themes.", "Global Styles", "wp_global_styles", nil, false, []string{}, "global-styles"),
		"wp_navigation": mk("Navigation menus that can be inserted into your site.", "Navigation Menus", "wp_navigation", nil, false, []string{}, "navigation"),
		"wp_font_family": mk("", "Font Families", "wp_font_family", nil, false, []string{}, "font-families"),
		"wp_font_face": mk("", "Font Faces", "wp_font_face", nil, false, []string{}, "font-families/(?P<font_family_id>[\\d]+)/font-faces"),
	}
}
