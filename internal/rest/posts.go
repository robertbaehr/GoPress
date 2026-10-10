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
	Rendered string `json:"rendered"`
}

// ProtectedRendered is content/excerpt with a protected flag.
type ProtectedRendered struct {
	Rendered  string `json:"rendered"`
	Protected bool   `json:"protected"`
}

// Post is a minimal wp/v2 post object for collection responses.
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
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	perPage := 10
	if v := r.URL.Query().Get("per_page"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			perPage = n
		}
		if perPage > 100 {
			perPage = 100
		}
	}
	posts, err := listPosts(c, perPage)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_post_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func listPosts(c *wp.Context, limit int) ([]Post, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 10
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_date, post_name, post_status, post_type, post_title, post_content, post_excerpt FROM "+
			c.DB.Posts()+" WHERE post_type = %s AND post_status = %s ORDER BY post_date DESC LIMIT %d",
		"post", "publish", limit,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]Post, 0, len(rows))
	for _, row := range rows {
		id := asInt64(row["ID"])
		dateStr := asString(row["post_date"])
		slug := asString(row["post_name"])
		title := asString(row["post_title"])
		content := asString(row["post_content"])
		excerpt := asString(row["post_excerpt"])
		out = append(out, Post{
			ID:     id,
			Date:   mysqlDateToREST(dateStr),
			Slug:   slug,
			Status: asString(row["post_status"]),
			Type:   asString(row["post_type"]),
			Link:   postPermalink(home, dateStr, slug, id),
			Title:  RenderedString{Rendered: title},
			Content: ProtectedRendered{
				Rendered:  content,
				Protected: false,
			},
			Excerpt: ProtectedRendered{
				Rendered:  excerpt,
				Protected: false,
			},
		})
	}
	return out, nil
}

func postPermalink(home, mysqlDate, slug string, id int64) string {
	if home == "" {
		home = ""
	}
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
