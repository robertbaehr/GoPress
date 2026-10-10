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

// Post is a wp/v2 post/page object for collection and item responses.
type Post struct {
	ID            int64             `json:"id"`
	Date          string            `json:"date"`
	DateGMT       string            `json:"date_gmt,omitempty"`
	GUID          RenderedString    `json:"guid"`
	Modified      string            `json:"modified"`
	ModifiedGMT   string            `json:"modified_gmt"`
	Slug          string            `json:"slug"`
	Status        string            `json:"status"`
	Type          string            `json:"type"`
	Link          string            `json:"link"`
	Title         RenderedString    `json:"title"`
	Content       ProtectedRendered `json:"content"`
	Excerpt       ProtectedRendered `json:"excerpt"`
	Author        int64             `json:"author"`
	FeaturedMedia int64             `json:"featured_media"`
	CommentStatus string            `json:"comment_status"`
	PingStatus    string            `json:"ping_status"`
	Sticky        bool              `json:"sticky"`
	Template      string            `json:"template"`
	Format        string            `json:"format"`
	Meta          map[string]any    `json:"meta"`
	Categories    []int64           `json:"categories"`
	Tags          []int64           `json:"tags"`
}

func (s *Server) servePosts(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.servePostCollection(w, r, c, "post")
}

func (s *Server) servePages(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	s.servePostCollection(w, r, c, "page")
}

func (s *Server) servePostItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	s.serveTypedItem(w, r, c, "post", id)
}

func (s *Server) servePageItem(w http.ResponseWriter, r *http.Request, c *wp.Context, id int64) {
	s.serveTypedItem(w, r, c, "page", id)
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
	sticky := stickyPostIDs(c)
	out := make([]Post, 0, len(rows))
	for _, row := range rows {
		out = append(out, postFromWP(c, row, home, edit, sticky))
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
	p := postFromWP(c, *row, home, edit, stickyPostIDs(c))
	return &p, nil
}

func postFromWP(c *wp.Context, row wp.Post, home string, edit bool, sticky map[int64]bool) Post {
	protected := row.Password != ""
	p := Post{
		ID:          row.ID,
		Date:        mysqlDateToREST(row.Date),
		DateGMT:     mysqlDateToREST(row.DateGMT),
		GUID:        RenderedString{Rendered: row.GUID},
		Modified:    mysqlDateToREST(row.Modified),
		ModifiedGMT: mysqlDateToREST(row.ModifiedGMT),
		Slug:        row.Name,
		Status:      row.Status,
		Type:        row.Type,
		Link:        itemPermalink(home, row.Type, row.Date, row.Name, row.ID),
		Title:       RenderedString{Rendered: row.Title},
		Content: ProtectedRendered{
			Rendered:  row.Content,
			Protected: protected,
		},
		Excerpt: ProtectedRendered{
			Rendered:  row.Excerpt,
			Protected: protected,
		},
		Author:        row.Author,
		FeaturedMedia: featuredMediaID(c, row.ID),
		CommentStatus: row.CommentStatus,
		PingStatus:    row.PingStatus,
		Sticky:        sticky[row.ID],
		Template:      pageTemplate(c, row.ID),
		Format:        postFormat(c, row.ID, row.Type),
		Meta:          map[string]any{},
		Categories:    postTermIDs(c, row.ID, "category"),
		Tags:          postTermIDs(c, row.ID, "post_tag"),
	}
	if edit {
		p.Title.Raw = row.Title
		p.Content.Raw = row.Content
		p.Excerpt.Raw = row.Excerpt
	}
	return p
}

func stickyPostIDs(c *wp.Context) map[int64]bool {
	out := map[int64]bool{}
	if c == nil || c.Options == nil {
		return out
	}
	raw := optionString(c, "sticky_posts", "")
	if raw == "" || !strings.HasPrefix(raw, "a:") {
		return out
	}
	// sticky_posts is PHP-serialized a:N:{i:k;i:id;…}; take array values.
	body := raw
	if i := strings.Index(body, "{"); i >= 0 {
		body = body[i+1:]
	}
	body = strings.TrimSuffix(strings.TrimSpace(body), "}")
	parts := strings.Split(body, ";")
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tokens = append(tokens, p)
		}
	}
	for i := 1; i < len(tokens); i += 2 {
		tok := tokens[i]
		if !strings.HasPrefix(tok, "i:") {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimPrefix(tok, "i:"), 10, 64)
		if err == nil && n > 0 {
			out[n] = true
		}
	}
	return out
}

func featuredMediaID(c *wp.Context, postID int64) int64 {
	if c == nil || c.Meta == nil || postID <= 0 {
		return 0
	}
	v := c.Meta.GetMetadata(wp.MetaPost, postID, "_thumbnail_id", true)
	return asInt64(v)
}

func pageTemplate(c *wp.Context, postID int64) string {
	if c == nil || c.Meta == nil || postID <= 0 {
		return ""
	}
	v := asString(c.Meta.GetMetadata(wp.MetaPost, postID, "_wp_page_template", true))
	if v == "" || v == "default" {
		return ""
	}
	return v
}

func postFormat(c *wp.Context, postID int64, postType string) string {
	if postType != "post" {
		return "standard"
	}
	slugs := postTermSlugs(c, postID, "post_format")
	if len(slugs) == 0 {
		return "standard"
	}
	slug := slugs[0]
	return strings.TrimPrefix(slug, "post-format-")
}

func postTermIDs(c *wp.Context, postID int64, taxonomy string) []int64 {
	ids := []int64{}
	if c == nil || c.DB == nil || postID <= 0 || taxonomy == "" {
		return ids
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.term_id FROM "+c.DB.Terms()+" t "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
			"INNER JOIN "+c.DB.TermRelationships()+" tr ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
			"WHERE tr.object_id = %d AND tt.taxonomy = %s ORDER BY t.term_id ASC",
		postID, taxonomy,
	)
	if err != nil {
		return ids
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return ids
	}
	for _, row := range rows {
		id := asInt64(row["term_id"])
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func postTermSlugs(c *wp.Context, postID int64, taxonomy string) []string {
	out := []string{}
	if c == nil || c.DB == nil || postID <= 0 || taxonomy == "" {
		return out
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.slug FROM "+c.DB.Terms()+" t "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
			"INNER JOIN "+c.DB.TermRelationships()+" tr ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
			"WHERE tr.object_id = %d AND tt.taxonomy = %s ORDER BY t.term_id ASC",
		postID, taxonomy,
	)
	if err != nil {
		return out
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return out
	}
	for _, row := range rows {
		s := asString(row["slug"])
		if s != "" {
			out = append(out, s)
		}
	}
	return out
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
