package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/db"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// SearchResult is a minimal wp/v2 search result object.
type SearchResult struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
}

func (s *Server) serveSearch(w http.ResponseWriter, r *http.Request, c *wp.Context) {
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
	term := strings.TrimSpace(r.URL.Query().Get("search"))
	total, err := countSearch(c, term)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_search_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	results, err := listSearchPaged(c, term, perPage, page)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_search_query",
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
	writeJSON(w, http.StatusOK, results)
}

func listSearchPaged(c *wp.Context, term string, limit, page int) ([]SearchResult, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	like := "%" + db.EscapeLike(term) + "%"
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_date, post_name, post_title, post_type FROM "+c.DB.Posts()+
			" WHERE post_status = %s AND post_type IN (%s, %s) AND (post_title LIKE %s OR post_content LIKE %s) "+
			"ORDER BY post_date DESC LIMIT %d OFFSET %d",
		"publish", "post", "page", like, like, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]SearchResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, searchFromRow(row, home))
	}
	return out, nil
}

func countSearch(c *wp.Context, term string) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	like := "%" + db.EscapeLike(term) + "%"
	q, args, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.Posts()+
			" WHERE post_status = %s AND post_type IN (%s, %s) AND (post_title LIKE %s OR post_content LIKE %s)",
		"publish", "post", "page", like, like,
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

func searchFromRow(row map[string]any, home string) SearchResult {
	id := asInt64(row["ID"])
	postType := asString(row["post_type"])
	slug := asString(row["post_name"])
	dateStr := asString(row["post_date"])
	return SearchResult{
		ID:      id,
		Title:   asString(row["post_title"]),
		URL:     itemPermalink(home, postType, dateStr, slug, id),
		Type:    "post",
		Subtype: postType,
	}
}
