package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// NavigationMenu is a wp/v2 navigation object (wp_navigation post type, P4.17).
type NavigationMenu struct {
	ID      int64             `json:"id"`
	Date    string            `json:"date"`
	Slug    string            `json:"slug"`
	Status  string            `json:"status"`
	Type    string            `json:"type"`
	Link    string            `json:"link"`
	Title   RenderedString    `json:"title"`
	Content ProtectedRendered `json:"content"`
}

func (s *Server) serveNavigation(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	switch r.Method {
	case http.MethodGet:
		// continue
	case http.MethodPost:
		s.createNavigation(w, r, c)
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
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	total, err := countPosts(c, "wp_navigation", nil, "", 0, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	rows, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "wp_navigation",
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
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]NavigationMenu, 0, len(rows))
	for _, row := range rows {
		out = append(out, navigationFromWP(row, home))
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	w.Header().Set("Allow", "GET, POST")
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveNavigationItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	switch r.Method {
	case http.MethodGet:
		// continue below
	case http.MethodDelete:
		s.deleteNavigation(w, r, c, id)
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
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "wp_navigation", PostStatus: "publish"})
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
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
	writeJSON(w, http.StatusOK, navigationFromWP(*row, home))
}

func navigationFromWP(row wp.Post, home string) NavigationMenu {
	link := home
	if home != "" {
		link = home + "/?p=" + strconv.FormatInt(row.ID, 10)
	}
	titleRaw, contentRaw := row.Title, row.Content
	return NavigationMenu{
		ID:     row.ID,
		Date:   mysqlDateToREST(row.Date),
		Slug:   row.Name,
		Status: row.Status,
		Type:   "wp_navigation",
		Link:   link,
		Title: RenderedString{
			Rendered: wp.KsesPost(row.Title),
			Raw:      &titleRaw,
		},
		Content: ProtectedRendered{
			Rendered:  wp.KsesPost(row.Content),
			Raw:       &contentRaw,
			Protected: row.Password != "",
		},
	}
}
