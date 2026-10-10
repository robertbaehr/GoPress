package rest

import (
	"net/http"
	"strconv"

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
		out = append(out, classicMenuFromTerm(t))
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
	writeJSON(w, http.StatusOK, classicMenuFromTerm(*term))
}

func classicMenuFromTerm(t Term) ClassicMenu {
	return ClassicMenu{
		ID:          t.ID,
		Count:       t.Count,
		Description: t.Description,
		Name:        t.Name,
		Slug:        t.Slug,
		Taxonomy:    "nav_menu",
		AutoAdd:     false,
		Locations:   []any{},
	}
}

func (s *Server) serveMenuItems(w http.ResponseWriter, r *http.Request, c *wp.Context) {
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
	perPage := parsePerPage(r, 100)
	page := parsePage(r)
	total, err := countPosts(c, "nav_menu_item")
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
	// Classic theme menu locations are not registered in GoPress yet; return empty map.
	_ = c
	writeJSON(w, http.StatusOK, map[string]MenuLocation{})
}
