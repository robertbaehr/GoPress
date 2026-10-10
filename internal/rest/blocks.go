package rest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// ReusableBlock is a wp/v2 blocks object (wp_block post type, P5.9).
// WP omits title.rendered / content.rendered for patterns.
type ReusableBlock struct {
	ID      int64             `json:"id"`
	Date    string            `json:"date"`
	Slug    string            `json:"slug"`
	Status  string            `json:"status"`
	Type    string            `json:"type"`
	Link    string            `json:"link"`
	Title   RenderedString    `json:"title"`
	Content ProtectedRendered `json:"content"`
}

func (s *Server) serveBlocks(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_read",
			"message": "Sorry, you are not allowed to view blocks.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	_ = wp.EnsurePatterns(c)
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	total, err := countPosts(c, "wp_block", nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	rows, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     "wp_block",
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
	out := make([]ReusableBlock, 0, len(rows))
	for _, row := range rows {
		out = append(out, reusableFromWP(row, home))
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveBlockItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	if !canEditPosts(c) {
		status := http.StatusUnauthorized
		if c != nil && c.CurrentUser != nil {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]any{
			"code":    "rest_cannot_read",
			"message": "Sorry, you are not allowed to view blocks.",
			"data":    map[string]any{"status": status},
		})
		return
	}
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: "wp_block", PostStatus: "publish"})
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
	writeJSON(w, http.StatusOK, reusableFromWP(*row, home))
}

func reusableFromWP(row wp.Post, home string) ReusableBlock {
	link := home
	if home != "" {
		link = home + "/?p=" + strconv.FormatInt(row.ID, 10)
	}
	titleRaw, contentRaw := row.Title, row.Content
	return ReusableBlock{
		ID:     row.ID,
		Date:   mysqlDateToREST(row.Date),
		Slug:   row.Name,
		Status: row.Status,
		Type:   "wp_block",
		Link:   link,
		Title: RenderedString{
			Raw: &titleRaw,
		},
		Content: ProtectedRendered{
			Raw:       &contentRaw,
			Protected: row.Password != "",
		},
	}
}
