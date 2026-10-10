package rest

import (
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PostStatus is a wp/v2 statuses object (P4.58).
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
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
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
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_view",
			"message": "Sorry, you are not allowed to manage post statuses.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	edit := context == "edit" && canEditPosts(c)
	canManage := canEditPosts(c)
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
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
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
	writeJSON(w, http.StatusOK, prepareStatus(c, *found, edit))
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
