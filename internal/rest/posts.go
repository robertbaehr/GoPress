package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// RenderedString is a WordPress REST rendered field ({"rendered": "..."}).
type RenderedString struct {
	Raw      string `json:"raw,omitempty"`
	Rendered string `json:"rendered"`
}

// ProtectedRendered is content/excerpt with a protected flag.
type ProtectedRendered struct {
	Raw       string `json:"raw,omitempty"`
	Rendered  string `json:"rendered"`
	Protected bool   `json:"protected"`
}

// Post is a minimal wp/v2 post/page object for collection and item responses.
type Post struct {
	ID      int64             `json:"id"`
	Date    string            `json:"date"`
	Slug    string            `json:"slug"`
	Status  string            `json:"status"`
	Type    string            `json:"type"`
	Link    string            `json:"link"`
	Title   RenderedString    `json:"title"`
	Content ProtectedRendered `json:"content"`
	Excerpt ProtectedRendered `json:"excerpt"`
}

func (s *Server) servePosts(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.servePostCollection(w, r, c, "post")
}

func (s *Server) servePages(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.servePostCollection(w, r, c, "page")
}

func (s *Server) servePostCollection(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	edit := r.URL.Query().Get("context") == "edit" && canEditPosts(c)
	total, err := countPosts(c, postType)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	posts, err := listPostsPaged(c, postType, perPage, page, edit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	setCollectionHeaders(w, r, c, total, totalPages, page)
	if canEditPosts(c) {
		w.Header().Set("Allow", "GET, POST")
	}
	writeJSON(w, http.StatusOK, posts)
}

func (s *Server) servePostItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	s.serveTypedItem(w, r, c, "post", id)
}

func (s *Server) servePageItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	s.serveTypedItem(w, r, c, "page", id)
}

func (s *Server) serveTypedItem(w http.ResponseWriter, r *http.Request, c *wp.Context, postType string, id int64) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	edit := r.URL.Query().Get("context") == "edit" && canEditPosts(c)
	post, err := getPostByID(c, postType, id, edit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func canEditPosts(c *wp.Context) bool {
	if c == nil || c.CurrentUser == nil {
		return false
	}
	return c.UserCan(c.CurrentUser, "edit_posts")
}

func parsePerPage(r *http.Request, def int) int {
	perPage := def
	if v := r.URL.Query().Get("per_page"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			perPage = n
		}
		if perPage > 100 {
			perPage = 100
		}
	}
	return perPage
}

func parsePage(r *http.Request) int {
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			page = n
		}
	}
	return page
}

func listPostsPaged(c *wp.Context, postType string, limit, page int, edit bool) ([]Post, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	if postType == "" {
		postType = "post"
	}
	offset := (page - 1) * limit
	rows, err := wp.GetPosts(c, wp.QueryArgs{
		PostType:     postType,
		PostStatus:   "publish",
		PostsPerPage: limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]Post, 0, len(rows))
	for _, row := range rows {
		out = append(out, postFromWP(row, home, edit))
	}
	return out, nil
}

func countPosts(c *wp.Context, postType string) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	if postType == "" {
		postType = "post"
	}
	q, args, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.Posts()+" WHERE post_type = %s AND post_status = %s",
		postType, "publish",
	)
	if err != nil {
		return 0, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return int(asInt64(row["total"])), nil
}

func getPostByID(c *wp.Context, postType string, id int64, edit bool) (*Post, error) {
	if id <= 0 {
		return nil, nil
	}
	row, err := wp.GetPost(c, wp.QueryArgs{P: id, PostType: postType, PostStatus: "publish"})
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	p := postFromWP(*row, home, edit)
	return &p, nil
}

func postFromWP(row wp.Post, home string, edit bool) Post {
	p := Post{
		ID:     row.ID,
		Date:   mysqlDateToREST(row.Date),
		Slug:   row.Name,
		Status: row.Status,
		Type:   row.Type,
		Link:   itemPermalink(home, row.Type, row.Date, row.Name, row.ID),
		Title:  RenderedString{Rendered: row.Title},
		Content: ProtectedRendered{
			Rendered:  row.Content,
			Protected: false,
		},
		Excerpt: ProtectedRendered{
			Rendered:  row.Excerpt,
			Protected: false,
		},
	}
	if edit {
		p.Title.Raw = row.Title
		p.Content.Raw = row.Content
		p.Excerpt.Raw = row.Excerpt
	}
	return p
}

func itemPermalink(home, postType, mysqlDate, slug string, id int64) string {
	if postType == "page" {
		return pagePermalink(home, slug, id)
	}
	return postPermalink(home, mysqlDate, slug, id)
}

func pagePermalink(home, slug string, id int64) string {
	if slug == "" {
		return fmt.Sprintf("%s/?page_id=%d", home, id)
	}
	return fmt.Sprintf("%s/%s/", home, slug)
}

func postPermalink(home, mysqlDate, slug string, id int64) string {
	t, err := time.Parse("2006-01-02 15:04:05", mysqlDate)
	if err != nil || slug == "" {
		return fmt.Sprintf("%s/?p=%d", home, id)
	}
	return fmt.Sprintf("%s/%04d/%02d/%02d/%s/", home, t.Year(), int(t.Month()), t.Day(), slug)
}

func mysqlDateToREST(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return strings.ReplaceAll(s, " ", "T")
	}
	return t.Format("2006-01-02T15:04:05")
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case uint64:
		return int64(t)
	case []byte:
		n, _ := strconv.ParseInt(string(t), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return n
	}
}
