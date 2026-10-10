package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// ClassicMenu is a wp/v2 menus object (nav_menu taxonomy term, P4.17 read stub).
type ClassicMenu struct {
	ID          int64  `json:"id"`
	Count       int64  `json:"count"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Taxonomy    string `json:"taxonomy"`
	AutoAdd     bool   `json:"auto_add"`
	Locations   []any  `json:"locations"`
}

// ClassicMenuItem is a minimal wp/v2 menu-items object (nav_menu_item post).
type ClassicMenuItem struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Type   string `json:"type"`
	Parent int64  `json:"parent"`
	Menus  int64  `json:"menus"`
	URL    string `json:"url"`
}

// MenuLocation is a wp/v2 menu-locations entry.
type MenuLocation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Menu        int64  `json:"menu"`
}

func canViewMenus(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	if c.UserCan(c.CurrentUser, "edit_theme_options") {
		return true
	}
	return c.UserCan(c.CurrentUser, "edit_posts")
}

func writeMenusForbidden(w http.ResponseWriter, c *wp.Context) {
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	writeJSON(w, status, map[string]any{
		"code":    "rest_cannot_view",
		"message": "Sorry, you are not allowed to view menus.",
		"data":    map[string]any{"status": status},
	})
}

func (s *Server) serveMenus(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewMenus(c) {
		writeMenusForbidden(w, c)
		return
	}
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	total, err := countTerms(c, "nav_menu")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_term_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	terms, err := listTermsPaged(c, "nav_menu", perPage, page)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_term_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	out := make([]ClassicMenu, 0, len(terms))
	for _, t := range terms {
		out = append(out, classicMenuFromTerm(c, t))
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveMenuItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewMenus(c) {
		writeMenusForbidden(w, c)
		return
	}
	term, err := getTermByID(c, "nav_menu", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_term_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if term == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_term_invalid",
			"message": "Term does not exist.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	writeJSON(w, http.StatusOK, classicMenuFromTerm(c, *term))
}

func classicMenuFromTerm(c *wp.Context, t Term) ClassicMenu {
	locs := []any{}
	for name, menuID := range wp.GetNavMenuLocations(c) {
		if menuID == t.ID {
			locs = append(locs, name)
		}
	}
	return ClassicMenu{
		ID:          t.ID,
		Count:       t.Count,
		Description: t.Description,
		Name:        t.Name,
		Slug:        t.Slug,
		Taxonomy:    "nav_menu",
		AutoAdd:     false,
		Locations:   locs,
	}
}

func (s *Server) serveMenuItems(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	switch r.Method {
	case http.MethodGet:
		// continue below
	case http.MethodPost:
		s.createMenuItem(w, r, c)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewMenus(c) {
		writeMenusForbidden(w, c)
		return
	}
	perPage := parsePerPage(r, 100)
	page := parsePage(r)
	total, err := countPosts(c, "nav_menu_item", nil, "", 0, nil, nil, nil, nil, nil, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	rows, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "nav_menu_item",
		PostStatus:   "publish",
		PostsPerPage: perPage,
		Offset:       (page - 1) * perPage,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	out := make([]ClassicMenuItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, classicMenuItemFromWP(c, row))
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveMenuItemByID(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	switch r.Method {
	case http.MethodGet:
		// continue below
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		s.updateMenuItem(w, r, c, id)
		return
	case http.MethodDelete:
		s.deleteMenuItem(w, r, c, id)
		return
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewMenus(c) {
		writeMenusForbidden(w, c)
		return
	}
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "nav_menu_item", PostStatus: "publish"})
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
	writeJSON(w, http.StatusOK, classicMenuItemFromWP(c, *row))
}

func classicMenuItemFromWP(c *wp.Context, row wp.Post) ClassicMenuItem {
	url := ""
	parent := row.Parent
	menuID := int64(0)
	if c != nil && c.Meta != nil {
		if v, ok := c.Meta.GetMetadata(wp.MetaPost, row.ID, "_menu_item_url", true).(string); ok {
			url = v
		}
		if v, ok := c.Meta.GetMetadata(wp.MetaPost, row.ID, "_menu_item_menu_item_parent", true).(string); ok && v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				parent = n
			}
		}
	}
	if c != nil && c.DB != nil {
		q, args, err := c.DB.Prepare(
			"SELECT tt.term_id FROM "+c.DB.TermRelationships()+" tr "+
				"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
				"WHERE tr.object_id = %d AND tt.taxonomy = %s LIMIT 1",
			row.ID, "nav_menu",
		)
		if err == nil {
			if r, err := c.DB.GetRow(q, args...); err == nil && r != nil {
				menuID = asInt64(r["term_id"])
			}
		}
	}
	return ClassicMenuItem{
		ID:     row.ID,
		Title:  row.Title,
		Status: row.Status,
		Type:   "nav_menu_item",
		Parent: parent,
		Menus:  menuID,
		URL:    url,
	}
}

func (s *Server) serveMenuLocations(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canViewMenus(c) {
		writeMenusForbidden(w, c)
		return
	}
	assigned := wp.GetNavMenuLocations(c)
	out := map[string]MenuLocation{}
	for name, desc := range wp.RegisteredNavMenus(c) {
		out[name] = MenuLocation{
			Name:        name,
			Description: desc,
			Menu:        assigned[name],
		}
	}
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveMenuLocationItem(w http.ResponseWriter, r *http.Request, c *wp.Context, location string) {
	location = strings.TrimSpace(location)
	registered := wp.RegisteredNavMenus(c)
	desc, ok := registered[location]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_menu_location_invalid",
			"message": "Invalid menu location.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !canViewMenus(c) {
			writeMenusForbidden(w, c)
			return
		}
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
		writeJSON(w, http.StatusOK, MenuLocation{
			Name:        location,
			Description: desc,
			Menu:        wp.GetNavMenuLocations(c)[location],
		})
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		s.updateMenuLocation(w, r, c, location, desc)
	case http.MethodDelete:
		s.deleteMenuLocation(w, r, c, location, desc)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
	}
}

// deleteMenuLocation unsets the assigned menu for a theme location (P4.74).
func (s *Server) deleteMenuLocation(w http.ResponseWriter, r *http.Request, c *wp.Context, location, desc string) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit menu locations.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_theme_options") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit menu locations.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	if err := wp.SetNavMenuLocation(c, location, 0); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_menu_location_invalid",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, MenuLocation{
		Name:        location,
		Description: desc,
		Menu:        0,
	})
}

func (s *Server) updateMenuLocation(w http.ResponseWriter, r *http.Request, c *wp.Context, location, desc string) {
	if c == nil || c.CurrentUser == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit menu locations.",
			"data":    map[string]any{"status": 401},
		})
		return
	}
	if !c.UserCan(c.CurrentUser, "edit_theme_options") {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"code":    "rest_cannot_edit",
			"message": "Sorry, you are not allowed to edit menu locations.",
			"data":    map[string]any{"status": 403},
		})
		return
	}
	menuID, err := decodeMenuLocationMenuID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	if err := wp.SetNavMenuLocation(c, location, menuID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_menu_location_invalid",
			"message": err.Error(),
			"data":    map[string]any{"status": 400},
		})
		return
	}
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, MenuLocation{
		Name:        location,
		Description: desc,
		Menu:        wp.GetNavMenuLocations(c)[location],
	})
}

func decodeMenuLocationMenuID(r *http.Request) (int64, error) {
	if r == nil || r.Body == nil {
		return 0, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return 0, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return 0, nil
	}
	var body struct {
		Menu any `json:"menu"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return 0, fmt.Errorf("invalid JSON body")
	}
	switch t := body.Menu.(type) {
	case nil:
		return 0, nil
	case float64:
		return int64(t), nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid menu id")
		}
		return n, nil
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0, fmt.Errorf("invalid menu id")
		}
		return n, nil
	default:
		n, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(t)), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid menu id")
		}
		return n, nil
	}
}
