package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// Media is a minimal wp/v2 media (attachment) object.
type Media struct {
	ID            int64          `json:"id"`
	Date          string         `json:"date"`
	Slug          string         `json:"slug"`
	Status        string         `json:"status"`
	Type          string         `json:"type"`
	Link          string         `json:"link"`
	Title         RenderedString `json:"title"`
	Author        int64          `json:"author"`
	CommentStatus string         `json:"comment_status"`
	PingStatus    string         `json:"ping_status"`
	MimeType      string         `json:"mime_type"`
	MediaType     string         `json:"media_type"`
	SourceURL     string         `json:"source_url"`
	GUID          RenderedString `json:"guid"`
}

func (s *Server) serveMedia(w http.ResponseWriter, r *http.Request, c *wp.Context) {
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
	total, err := countMedia(c)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_media_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	items, err := listMediaPaged(c, perPage, page)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_media_query",
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
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) serveMediaItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"code":    "rest_no_route",
			"message": "No route was found matching the URL and request method.",
			"data":    map[string]any{"status": 405},
		})
		return
	}
	item, err := getMediaByID(c, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code":    "rest_media_query",
			"message": err.Error(),
			"data":    map[string]any{"status": 500},
		})
		return
	}
	if item == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"code":    "rest_post_invalid_id",
			"message": "Invalid post ID.",
			"data":    map[string]any{"status": 404},
		})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func listMediaPaged(c *wp.Context, limit, page int) ([]Media, error) {
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
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_name, post_status, post_title, post_mime_type, guid, "+
			"comment_status, ping_status FROM "+c.DB.Posts()+
			" WHERE post_type = %s AND post_status = %s ORDER BY post_date DESC LIMIT %d OFFSET %d",
		"attachment", "inherit", limit, offset,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]Media, 0, len(rows))
	for _, row := range rows {
		out = append(out, mediaFromRow(row, home))
	}
	return out, nil
}

func countMedia(c *wp.Context) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	q, args, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.Posts()+" WHERE post_type = %s AND post_status = %s",
		"attachment", "inherit",
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

func getMediaByID(c *wp.Context, id int64) (*Media, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_name, post_status, post_title, post_mime_type, guid, "+
			"comment_status, ping_status FROM "+c.DB.Posts()+
			" WHERE ID = %d AND post_type = %s AND post_status = %s LIMIT 1",
		id, "attachment", "inherit",
	)
	if err != nil {
		return nil, err
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	m := mediaFromRow(row, home)
	return &m, nil
}

func mediaFromRow(row map[string]any, home string) Media {
	id := asInt64(row["ID"])
	slug := asString(row["post_name"])
	guid := asString(row["guid"])
	mime := asString(row["post_mime_type"])
	title := asString(row["post_title"])
	return Media{
		ID:            id,
		Date:          mysqlDateToREST(asString(row["post_date"])),
		Slug:          slug,
		Status:        asString(row["post_status"]),
		Type:          "attachment",
		Link:          mediaPermalink(home, slug, id),
		Title:         RenderedString{Rendered: title},
		Author:        asInt64(row["post_author"]),
		CommentStatus: asString(row["comment_status"]),
		PingStatus:    asString(row["ping_status"]),
		MimeType:      mime,
		MediaType:     mediaTypeFromMime(mime),
		SourceURL:     sourceURLFromGUID(guid, home),
		GUID:          RenderedString{Rendered: guid},
	}
}

func mediaPermalink(home, slug string, id int64) string {
	if slug == "" {
		return fmt.Sprintf("%s/?attachment_id=%d", home, id)
	}
	return fmt.Sprintf("%s/%s/", home, slug)
}

func mediaTypeFromMime(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	default:
		return "file"
	}
}

// sourceURLFromGUID prefers the post guid when it looks like a URL; otherwise
// best-effort builds an uploads path under home.
func sourceURLFromGUID(guid, home string) string {
	guid = strings.TrimSpace(guid)
	if strings.HasPrefix(guid, "http://") || strings.HasPrefix(guid, "https://") {
		return guid
	}
	if guid == "" {
		return ""
	}
	if strings.HasPrefix(guid, "/") {
		return home + guid
	}
	if strings.Contains(guid, "wp-content/uploads/") {
		return home + "/" + strings.TrimPrefix(guid, "/")
	}
	return home + "/wp-content/uploads/" + strings.TrimPrefix(guid, "/")
}
