package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PostTypeVisibility is the edit-context visibility object (P4.125).
type PostTypeVisibility struct {
	ShowInNavMenus bool `json:"show_in_nav_menus"`
	ShowUI         bool `json:"show_ui"`
}

// PostType is a wp/v2 type object (view context; edit fields omitempty — P4.125).
type PostType struct {
	Capabilities  map[string]any      `json:"capabilities,omitempty"`
	Description   string              `json:"description"`
	Hierarchical  bool                `json:"hierarchical"`
	Viewable      *bool               `json:"viewable,omitempty"`
	Labels        map[string]any      `json:"labels,omitempty"`
	Name          string              `json:"name"`
	Slug          string              `json:"slug"`
	Supports      map[string]any      `json:"supports,omitempty"`
	HasArchive    any                 `json:"has_archive"`
	Taxonomies    []string            `json:"taxonomies"`
	RestBase      string              `json:"rest_base"`
	RestNamespace string              `json:"rest_namespace"`
	Visibility    *PostTypeVisibility `json:"visibility,omitempty"`
	Icon          any                 `json:"icon"`
	Template      []any               `json:"template"`
	TemplateLock  any                 `json:"template_lock"`
	Links         map[string]any      `json:"_links"`
}

func (s *Server) serveTypes(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P4.125: OPTIONS returns public item schema.
	// Allow header matches WP oracle (READABLE → "GET") even though HEAD/OPTIONS work.
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, postTypeItemSchema())
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !validatePostTypeContext(w, r) {
		return
	}
	context := strings.TrimSpace(r.URL.Query().Get("context"))
	if context == "" {
		context = "view"
	}
	edit := context == "edit"
	if edit && !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		w.Header().Set("Allow", "GET")
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view",
			"message": "Sorry, you are not allowed to edit posts in this post type.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	w.Header().Set("Allow", "GET")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, builtinPostTypes(c, edit))
}

// serveTypeItem handles GET/HEAD/OPTIONS /wp/v2/types/{type} (P4.125).
func (s *Server) serveTypeItem(w http.ResponseWriter, r *http.Request, c *wp.Context, slug string) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, postTypeItemSchema())
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !validatePostTypeContext(w, r) {
		return
	}
	slug = strings.TrimSpace(slug)
	if slug == "" || strings.Contains(slug, "/") || strings.Contains(slug, "..") {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_type_invalid",
			"message": "Invalid post type.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	context := strings.TrimSpace(r.URL.Query().Get("context"))
	if context == "" {
		context = "view"
	}
	edit := context == "edit"
	if edit && !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		w.Header().Set("Allow", "GET")
		writeJSON(w, status, map[string]any{
			"code":    "rest_forbidden_context",
			"message": "Sorry, you are not allowed to edit posts in this post type.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	all := builtinPostTypes(c, edit)
	pt, ok := all[slug]
	if !ok {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_type_invalid",
			"message": "Invalid post type.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	w.Header().Set("Allow", "GET")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("rest_prepare_post_type", pt, slug, r)
		if m, ok := filtered.(PostType); ok {
			pt = m
		}
	}
	writeJSON(w, http.StatusOK, pt)
}

func builtinPostTypes(c *wp.Context, edit bool) map[string]PostType {
	base := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	restRoot := base + "/wp-json"
	typesBase := restRoot + "/wp/v2/types"

	mk := func(description, name, slug string, icon any, hierarchical bool, taxonomies []string, restBase string, viewable bool, showUI bool, supports map[string]any, labels map[string]any, caps map[string]any) PostType {
		if taxonomies == nil {
			taxonomies = []string{}
		}
		pt := PostType{
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
		if edit {
			v := viewable
			pt.Viewable = &v
			pt.Visibility = &PostTypeVisibility{
				ShowInNavMenus: showUI,
				ShowUI:         showUI,
			}
			pt.Supports = supports
			pt.Labels = labels
			pt.Capabilities = caps
		}
		return pt
	}

	postSupports := map[string]any{
		"title": true, "editor": true, "author": true, "thumbnail": true,
		"excerpt": true, "trackbacks": true, "custom-fields": true,
		"comments": true, "revisions": true, "post-formats": true,
	}
	pageSupports := map[string]any{
		"title": true, "editor": true, "author": true, "thumbnail": true,
		"page-attributes": true, "custom-fields": true, "comments": true, "revisions": true,
	}
	mediaSupports := map[string]any{
		"title": true, "author": true, "comments": true,
	}

	out := map[string]PostType{
		"post": mk("", "Posts", "post", "dashicons-admin-post", false, []string{"category", "post_tag"}, "posts", true, true, postSupports,
			map[string]any{"name": "Posts", "singular_name": "Post"},
			map[string]any{"edit_posts": "edit_posts", "publish_posts": "publish_posts"}),
		"page": mk("", "Pages", "page", "dashicons-admin-page", true, []string{}, "pages", true, true, pageSupports,
			map[string]any{"name": "Pages", "singular_name": "Page"},
			map[string]any{"edit_posts": "edit_pages", "publish_posts": "publish_pages"}),
		"attachment": mk("", "Media", "attachment", "dashicons-admin-media", false, []string{}, "media", true, true, mediaSupports,
			map[string]any{"name": "Media", "singular_name": "Media"},
			map[string]any{"edit_posts": "edit_posts", "create_posts": "upload_files"}),
		"nav_menu_item": mk("", "Navigation Menu Items", "nav_menu_item", nil, false, []string{"nav_menu"}, "menu-items", false, false,
			map[string]any{"title": true, "custom-fields": true},
			map[string]any{"name": "Navigation Menu Items", "singular_name": "Navigation Menu Item"},
			map[string]any{"edit_posts": "edit_theme_options"}),
		"wp_block": mk("", "Patterns", "wp_block", nil, false, []string{"wp_pattern_category"}, "blocks", false, true,
			map[string]any{"title": true, "editor": true, "custom-fields": true, "revisions": true},
			map[string]any{"name": "Patterns", "singular_name": "Pattern"},
			map[string]any{"edit_posts": "edit_posts"}),
		"wp_template": mk("Templates to include in your theme.", "Templates", "wp_template", nil, false, []string{}, "templates", false, false,
			map[string]any{"title": true, "editor": true, "revisions": true},
			map[string]any{"name": "Templates", "singular_name": "Template"},
			map[string]any{"edit_posts": "edit_theme_options"}),
		"wp_template_part": mk("Template parts to include in your templates.", "Template Parts", "wp_template_part", nil, false, []string{}, "template-parts", false, false,
			map[string]any{"title": true, "editor": true, "revisions": true},
			map[string]any{"name": "Template Parts", "singular_name": "Template Part"},
			map[string]any{"edit_posts": "edit_theme_options"}),
		"wp_global_styles": mk("Global styles to include in themes.", "Global Styles", "wp_global_styles", nil, false, []string{}, "global-styles", false, false,
			map[string]any{"title": true, "editor": true, "revisions": true},
			map[string]any{"name": "Global Styles", "singular_name": "Global Styles"},
			map[string]any{"edit_posts": "edit_theme_options"}),
		"wp_navigation": mk("Navigation menus that can be inserted into your site.", "Navigation Menus", "wp_navigation", nil, false, []string{}, "navigation", false, false,
			map[string]any{"title": true, "editor": true, "revisions": true},
			map[string]any{"name": "Navigation Menus", "singular_name": "Navigation Menu"},
			map[string]any{"edit_posts": "edit_theme_options"}),
		"wp_font_family": mk("", "Font Families", "wp_font_family", nil, false, []string{}, "font-families", false, false,
			map[string]any{},
			map[string]any{"name": "Font Families", "singular_name": "Font Family"},
			map[string]any{"edit_posts": "edit_theme_options"}),
		"wp_font_face": mk("", "Font Faces", "wp_font_face", nil, false, []string{}, "font-families/(?P<font_family_id>[\\d]+)/font-faces", false, false,
			map[string]any{},
			map[string]any{"name": "Font Faces", "singular_name": "Font Face"},
			map[string]any{"edit_posts": "edit_theme_options"}),
	}
	return out
}

func validatePostTypeContext(w http.ResponseWriter, r *http.Request) bool {
	ctx := strings.TrimSpace(r.URL.Query().Get("context"))
	if ctx == "" {
		return true
	}
	switch ctx {
	case "view", "embed", "edit":
		return true
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): context",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{
					"context": "context is not one of view, embed, edit.",
				},
			},
		})
		return false
	}
}

// postTypeItemSchema is the public REST schema for /wp/v2/types (P4.125).
func postTypeItemSchema() map[string]any {
	return map[string]any{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"title":   "type",
		"type":    "object",
		"properties": map[string]any{
			"capabilities":   map[string]any{"type": "object", "context": []any{"edit"}, "readonly": true},
			"description":    map[string]any{"type": "string", "context": []any{"view", "edit"}, "readonly": true},
			"hierarchical":   map[string]any{"type": "boolean", "context": []any{"view", "edit"}, "readonly": true},
			"viewable":       map[string]any{"type": "boolean", "context": []any{"edit"}, "readonly": true},
			"labels":         map[string]any{"type": "object", "context": []any{"edit"}, "readonly": true},
			"name":           map[string]any{"type": "string", "context": []any{"view", "edit", "embed"}, "readonly": true},
			"slug":           map[string]any{"type": "string", "context": []any{"view", "edit", "embed"}, "readonly": true},
			"supports":       map[string]any{"type": "object", "context": []any{"edit"}, "readonly": true},
			"has_archive":    map[string]any{"type": []any{"string", "boolean"}, "context": []any{"view", "edit"}, "readonly": true},
			"taxonomies":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "context": []any{"view", "edit"}, "readonly": true},
			"rest_base":      map[string]any{"type": "string", "context": []any{"view", "edit", "embed"}, "readonly": true},
			"rest_namespace": map[string]any{"type": "string", "context": []any{"view", "edit", "embed"}, "readonly": true},
			"visibility":     map[string]any{"type": "object", "context": []any{"edit"}, "readonly": true},
			"icon":           map[string]any{"type": []any{"string", "null"}, "context": []any{"view", "edit"}, "readonly": true},
			"template":       map[string]any{"type": "array", "context": []any{"view", "edit"}, "readonly": true},
			"template_lock":  map[string]any{"type": []any{"string", "boolean"}, "context": []any{"view", "edit"}, "readonly": true},
		},
	}
}
