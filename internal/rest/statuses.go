package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PostStatus is a wp/v2 statuses object (P4.58 / P4.126).
type PostStatus struct {
	Name         string         `json:"name"`
	Private      *bool          `json:"private,omitempty"`
	Protected    *bool          `json:"protected,omitempty"`
	Public       bool           `json:"public"`
	Queryable    bool           `json:"queryable"`
	ShowInList   *bool          `json:"show_in_list,omitempty"`
	Slug         string         `json:"slug"`
	DateFloating bool           `json:"date_floating"`
	Links        map[string]any `json:"_links"`
}

type statusDef struct {
	Label        string
	Slug         string
	Public       bool
	Queryable    bool
	Private      bool
	Protected    bool
	ShowInList   bool
	DateFloating bool
	Internal     bool
}

func builtinStatusDefs() []statusDef {
	return []statusDef{
		{Label: "Published", Slug: "publish", Public: true, Queryable: true, ShowInList: true},
		{Label: "Scheduled", Slug: "future", Protected: true, Queryable: true, ShowInList: true},
		{Label: "Draft", Slug: "draft", Protected: true, ShowInList: true, DateFloating: true},
		{Label: "Pending", Slug: "pending", Protected: true, ShowInList: true, DateFloating: true},
		{Label: "Private", Slug: "private", Private: true, Queryable: true, ShowInList: true},
		{Label: "Trash", Slug: "trash", ShowInList: false, Internal: true}, // included explicitly like WP
	}
}

func (s *Server) serveStatuses(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P4.126: OPTIONS returns public item schema; Allow matches WP oracle (GET).
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, postStatusItemSchema())
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
	if !validatePostStatusContext(w, r) {
		return
	}
	context := r.URL.Query().Get("context")
	if context == "" {
		context = "view"
	}
	if context == "edit" && !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		// Oracle omits Allow on auth failures for statuses collection.
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view",
			"message": "Sorry, you are not allowed to manage post statuses.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	edit := context == "edit" && canEditPosts(c)
	canManage := canEditPosts(c)
	w.Header().Set("Allow", "GET")
	if r.Method == http.MethodHead {
		// Match WP: HEAD still advertises JSON content type.
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		return
	}
	out := map[string]PostStatus{}
	for _, def := range builtinStatusDefs() {
		if !statusReadable(def, canManage) {
			continue
		}
		out[def.Slug] = prepareStatus(c, def, edit)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveStatusItem(w http.ResponseWriter, r *http.Request, c *wp.Context, slug string) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusOK, postStatusItemSchema())
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
	if !validatePostStatusContext(w, r) {
		return
	}
	slug = strings.TrimSpace(strings.ToLower(slug))
	var found *statusDef
	for i := range builtinStatusDefs() {
		d := builtinStatusDefs()[i]
		if d.Slug == slug {
			found = &d
			break
		}
	}
	if found == nil {
		// Oracle omits Allow on invalid status 404s.
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_status_invalid",
			"message": "Invalid status.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	context := r.URL.Query().Get("context")
	if context == "" {
		context = "view"
	}
	canManage := canEditPosts(c)
	if !statusReadable(*found, canManage) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		// Oracle omits Allow on unreadable status auth failures.
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_read_status",
			"message": "Cannot view status.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	if context == "edit" && !canManage {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_read_status",
			"message": "Cannot view status.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	edit := context == "edit" && canManage
	w.Header().Set("Allow", "GET")
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		return
	}
	st := prepareStatus(c, *found, edit)
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("rest_prepare_status", st, found.Slug, r)
		if m, ok := filtered.(PostStatus); ok {
			st = m
		}
	}
	writeJSON(w, http.StatusOK, st)
}

func statusReadable(def statusDef, canManage bool) bool {
	if def.Public {
		return true
	}
	// Non-public / trash visible only to users who can edit posts (WP check_read_permission).
	return canManage
}

func prepareStatus(c *wp.Context, def statusDef, edit bool) PostStatus {
	base := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	archives := base + "/wp-json/wp/v2/posts"
	if def.Slug != "publish" {
		archives += "?status=" + def.Slug
	}
	st := PostStatus{
		Name:         def.Label,
		Public:       def.Public,
		Queryable:    def.Queryable,
		Slug:         def.Slug,
		DateFloating: def.DateFloating,
		Links: map[string]any{
			"archives": []map[string]any{{"href": archives}},
		},
	}
	if edit {
		p, pr, show := def.Private, def.Protected, def.ShowInList
		st.Private = &p
		st.Protected = &pr
		st.ShowInList = &show
	}
	return st
}

func validatePostStatusContext(w http.ResponseWriter, r *http.Request) bool {
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

// postStatusItemSchema is the public REST schema for /wp/v2/statuses (P4.126).
func postStatusItemSchema() map[string]any {
	return map[string]any{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"title":   "status",
		"type":    "object",
		"properties": map[string]any{
			"name":          map[string]any{"description": "The title for the status.", "type": "string", "context": []any{"embed", "view", "edit"}, "readonly": true},
			"private":       map[string]any{"description": "Whether posts with this status should be private.", "type": "boolean", "context": []any{"edit"}, "readonly": true},
			"protected":     map[string]any{"description": "Whether posts with this status should be protected.", "type": "boolean", "context": []any{"edit"}, "readonly": true},
			"public":        map[string]any{"description": "Whether posts of this status should be shown in the front end of the site.", "type": "boolean", "context": []any{"view", "edit"}, "readonly": true},
			"queryable":     map[string]any{"description": "Whether posts with this status should be publicly-queryable.", "type": "boolean", "context": []any{"view", "edit"}, "readonly": true},
			"show_in_list":  map[string]any{"description": "Whether to include posts in the edit listing for their post type.", "type": "boolean", "context": []any{"edit"}, "readonly": true},
			"slug":          map[string]any{"description": "An alphanumeric identifier for the status.", "type": "string", "context": []any{"embed", "view", "edit"}, "readonly": true},
			"date_floating": map[string]any{"description": "Whether posts of this status may have floating published dates.", "type": "boolean", "context": []any{"view", "edit"}, "readonly": true},
		},
	}
}
