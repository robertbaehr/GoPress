package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// RenderedString is a WordPress REST rendered field ({"rendered": "..."}).
// Raw is a pointer so edit-context can emit "" while view omits the key.
type RenderedString struct {
	Raw      *string `json:"raw,omitempty"`
	Rendered string  `json:"rendered"`
}

// ProtectedRendered is content/excerpt with a protected flag.
type ProtectedRendered struct {
	Raw          *string `json:"raw,omitempty"`
	Rendered     string  `json:"rendered"`
	Protected    bool    `json:"protected"`
	BlockVersion *int    `json:"block_version,omitempty"`
}

// Post is a wp/v2 post/page object for collection and item responses.
type Post struct {
	ID            int64             `json:"id"`
	Date          string            `json:"date"`
	DateGMT       string            `json:"date_gmt"`
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
	Parent        int64             `json:"parent"`
	MenuOrder     int               `json:"menu_order"`
	ClassList     []string          `json:"class_list"`
	Links         map[string]any    `json:"_links"`
	// Edit-context fields (nil/absent in view; pointer so "" is emitted when set).
	Password          *string `json:"password,omitempty"`
	PermalinkTemplate string  `json:"permalink_template,omitempty"`
	GeneratedSlug     string  `json:"generated_slug,omitempty"`
}

// MarshalJSON omits post-only fields on pages and page-only fields on posts.
func (p Post) MarshalJSON() ([]byte, error) {
	if p.Type == "page" {
		type pageOut struct {
			ID                int64             `json:"id"`
			Date              string            `json:"date"`
			DateGMT           string            `json:"date_gmt"`
			GUID              RenderedString    `json:"guid"`
			Modified          string            `json:"modified"`
			ModifiedGMT       string            `json:"modified_gmt"`
			Slug              string            `json:"slug"`
			Status            string            `json:"status"`
			Type              string            `json:"type"`
			Link              string            `json:"link"`
			Title             RenderedString    `json:"title"`
			Content           ProtectedRendered `json:"content"`
			Excerpt           ProtectedRendered `json:"excerpt"`
			Author            int64             `json:"author"`
			FeaturedMedia     int64             `json:"featured_media"`
			Parent            int64             `json:"parent"`
			MenuOrder         int               `json:"menu_order"`
			CommentStatus     string            `json:"comment_status"`
			PingStatus        string            `json:"ping_status"`
			Template          string            `json:"template"`
			Meta              map[string]any    `json:"meta"`
			ClassList         []string          `json:"class_list"`
			Links             map[string]any    `json:"_links"`
			Password          *string           `json:"password,omitempty"`
			PermalinkTemplate string            `json:"permalink_template,omitempty"`
			GeneratedSlug     string            `json:"generated_slug,omitempty"`
		}
		return json.Marshal(pageOut{
			ID: p.ID, Date: p.Date, DateGMT: p.DateGMT, GUID: p.GUID,
			Modified: p.Modified, ModifiedGMT: p.ModifiedGMT, Slug: p.Slug,
			Status: p.Status, Type: p.Type, Link: p.Link, Title: p.Title,
			Content: p.Content, Excerpt: p.Excerpt, Author: p.Author,
			FeaturedMedia: p.FeaturedMedia, Parent: p.Parent, MenuOrder: p.MenuOrder,
			CommentStatus: p.CommentStatus, PingStatus: p.PingStatus,
			Template: p.Template, Meta: p.Meta, ClassList: p.ClassList, Links: p.Links,
			Password: p.Password, PermalinkTemplate: p.PermalinkTemplate,
			GeneratedSlug: p.GeneratedSlug,
		})
	}
	type postOut struct {
		ID                int64             `json:"id"`
		Date              string            `json:"date"`
		DateGMT           string            `json:"date_gmt"`
		GUID              RenderedString    `json:"guid"`
		Modified          string            `json:"modified"`
		ModifiedGMT       string            `json:"modified_gmt"`
		Slug              string            `json:"slug"`
		Status            string            `json:"status"`
		Type              string            `json:"type"`
		Link              string            `json:"link"`
		Title             RenderedString    `json:"title"`
		Content           ProtectedRendered `json:"content"`
		Excerpt           ProtectedRendered `json:"excerpt"`
		Author            int64             `json:"author"`
		FeaturedMedia     int64             `json:"featured_media"`
		CommentStatus     string            `json:"comment_status"`
		PingStatus        string            `json:"ping_status"`
		Sticky            bool              `json:"sticky"`
		Template          string            `json:"template"`
		Format            string            `json:"format"`
		Meta              map[string]any    `json:"meta"`
		Categories        []int64           `json:"categories"`
		Tags              []int64           `json:"tags"`
		ClassList         []string          `json:"class_list"`
		Links             map[string]any    `json:"_links"`
		Password          *string           `json:"password,omitempty"`
		PermalinkTemplate string            `json:"permalink_template,omitempty"`
		GeneratedSlug     string            `json:"generated_slug,omitempty"`
	}
	return json.Marshal(postOut{
		ID: p.ID, Date: p.Date, DateGMT: p.DateGMT, GUID: p.GUID,
		Modified: p.Modified, ModifiedGMT: p.ModifiedGMT, Slug: p.Slug,
		Status: p.Status, Type: p.Type, Link: p.Link, Title: p.Title,
		Content: p.Content, Excerpt: p.Excerpt, Author: p.Author,
		FeaturedMedia: p.FeaturedMedia, CommentStatus: p.CommentStatus,
		PingStatus: p.PingStatus, Sticky: p.Sticky, Template: p.Template,
		Format: p.Format, Meta: p.Meta, Categories: p.Categories, Tags: p.Tags,
		ClassList: p.ClassList, Links: p.Links,
		Password: p.Password, PermalinkTemplate: p.PermalinkTemplate,
		GeneratedSlug: p.GeneratedSlug,
	})
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

// parsePostsOffset returns REST ?offset= when present (P4.92).
// ok is false when the param is absent or non-numeric; negative values become 0.
func parsePostsOffset(r *http.Request) (offset int, ok bool) {
	v := strings.TrimSpace(r.URL.Query().Get("offset"))
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	if n < 0 {
		n = 0
	}
	return n, true
}

func listPostsPaged(c *wp.Context, postType string, limit, page int, edit bool, stickyFilter *bool, orderby, order, search string, author int64, categoryIn, tagIn []int64, statuses []string, slugs []string, include, exclude, parentIn, parentNotIn []int64, dateAfter, dateBefore, modifiedAfter, modifiedBefore string, formatIn []string, formatStandard bool, offsetOverride int, useOffset bool) ([]Post, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	if postType == "" {
		postType = "post"
	}
	// WP_Query: numeric offset takes precedence over paged (P4.92).
	offset := (page - 1) * limit
	if useOffset {
		offset = offsetOverride
	}
	args := wp.QueryArgs{
		PostType:       postType,
		PostStatus:     "publish",
		PostStatuses:   statuses,
		PostsPerPage:   limit,
		Offset:         offset,
		OrderBy:        orderby,
		Order:          order,
		Search:         search,
		Author:         author,
		CategoryIn:     categoryIn,
		TagIn:          tagIn,
		NameIn:         slugs,
		PostIn:         include,
		PostNotIn:      exclude,
		ParentIn:       parentIn,
		ParentNotIn:    parentNotIn,
		DateAfter:          dateAfter,
		DateBefore:         dateBefore,
		ModifiedAfter:      modifiedAfter,
		ModifiedBefore:     modifiedBefore,
		PostFormatIn:       formatIn,
		PostFormatStandard: formatStandard,
	}
	applyStickyQueryFilter(c, postType, stickyFilter, &args)
	rows, err := wp.GetPosts(c, args)
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

// parsePostsOrderby accepts REST orderby=title|date|id|slug|menu_order (P4.83 / P4.94).
// Default date.
func parsePostsOrderby(r *http.Request) string {
	v := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("orderby")))
	switch v {
	case "title", "date", "id", "slug", "menu_order":
		return v
	case "":
		return "date"
	default:
		return "date"
	}
}

// parsePostsOrder accepts REST order=asc|desc (P4.83). Default desc.
func parsePostsOrder(r *http.Request) string {
	v := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if v == "asc" {
		return "asc"
	}
	return "desc"
}

// parsePostsSearch reads REST ?search= (maps to WP_Query s / QueryArgs.Search, P4.84).
func parsePostsSearch(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get("search"))
}

// parsePostsAuthor reads REST ?author= (maps to WP_Query author / author__in, P4.85).
// Accepts a single integer or the first value of a repeated/comma list.
func parsePostsAuthor(r *http.Request) int64 {
	if r == nil {
		return 0
	}
	raw := strings.TrimSpace(r.URL.Query().Get("author"))
	if raw == "" {
		return 0
	}
	// WP allows author[]=1&author[]=2; net/http Get returns the first.
	if i := strings.IndexByte(raw, ','); i >= 0 {
		raw = strings.TrimSpace(raw[:i])
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// parsePostsCategories reads REST ?categories= (maps to tax_query category term_id IN, P4.86).
// Accepts comma-separated IDs and/or repeated query values.
func parsePostsCategories(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "categories")
}

// parsePostsTags reads REST ?tags= (maps to tax_query post_tag term_id IN, P4.87).
// Accepts comma-separated IDs and/or repeated query values.
func parsePostsTags(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "tags")
}

// knownPostFormats mirrors get_post_format_slugs() (P4.96).
var knownPostFormats = map[string]bool{
	"standard": true,
	"aside":    true,
	"chat":     true,
	"gallery":  true,
	"link":     true,
	"image":    true,
	"quote":    true,
	"status":   true,
	"video":    true,
	"audio":    true,
}

// parsePostsFormat reads REST ?format= (post_format tax_query, P4.96).
// Returns post-format-* slugs plus whether standard (NOT EXISTS) is requested.
func parsePostsFormat(r *http.Request) (formatIn []string, includeStandard bool) {
	if r == nil {
		return nil, false
	}
	q := r.URL.Query()
	vals, ok := q["format"]
	if !ok {
		return nil, false
	}
	seen := map[string]bool{}
	add := func(raw string) {
		raw = strings.ToLower(strings.TrimSpace(raw))
		if raw == "" || !knownPostFormats[raw] || seen[raw] {
			return
		}
		seen[raw] = true
		if raw == "standard" {
			includeStandard = true
			return
		}
		formatIn = append(formatIn, "post-format-"+raw)
	}
	for _, v := range vals {
		for _, part := range strings.Split(v, ",") {
			add(part)
		}
	}
	return formatIn, includeStandard
}

// parsePostsInclude reads REST ?include= (maps to post__in, P4.90).
func parsePostsInclude(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "include")
}

// parsePostsExclude reads REST ?exclude= (maps to post__not_in, P4.90).
func parsePostsExclude(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "exclude")
}

// parsePostsParent reads REST ?parent= (maps to post_parent__in, P4.93).
// Allows 0 for top-level hierarchical posts/pages. nil when absent.
func parsePostsParent(r *http.Request) []int64 {
	return parsePostsParentIDs(r, "parent")
}

// parsePostsParentExclude reads REST ?parent_exclude= (post_parent__not_in, P4.93).
func parsePostsParentExclude(r *http.Request) []int64 {
	return parsePostsParentIDs(r, "parent_exclude")
}

// parsePostsParentIDs parses parent / parent_exclude, allowing 0.
// Returns nil when the query key is absent; non-nil (possibly empty) when present.
func parsePostsParentIDs(r *http.Request, key string) []int64 {
	if r == nil || key == "" {
		return nil
	}
	q := r.URL.Query()
	vals, ok := q[key]
	if !ok {
		return nil
	}
	seen := map[int64]bool{}
	var out []int64
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.ParseInt(part, 10, 64)
			if err != nil || n < 0 || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	for _, v := range vals {
		add(v)
	}
	if out == nil {
		out = []int64{}
	}
	return out
}

// parsePostsAfter reads REST ?after= (ISO8601 → MySQL datetime, P4.91).
func parsePostsAfter(r *http.Request) string {
	return parseRESTDateTimeQuery(r, "after")
}

// parsePostsBefore reads REST ?before= (ISO8601 → MySQL datetime, P4.91).
func parsePostsBefore(r *http.Request) string {
	return parseRESTDateTimeQuery(r, "before")
}

// parsePostsModifiedAfter reads REST ?modified_after= (P4.95).
func parsePostsModifiedAfter(r *http.Request) string {
	return parseRESTDateTimeQuery(r, "modified_after")
}

// parsePostsModifiedBefore reads REST ?modified_before= (P4.95).
func parsePostsModifiedBefore(r *http.Request) string {
	return parseRESTDateTimeQuery(r, "modified_before")
}

// parseRESTDateTimeQuery parses an ISO8601 / MySQL datetime query arg to MySQL form.
func parseRESTDateTimeQuery(r *http.Request, key string) string {
	if r == nil || key == "" {
		return ""
	}
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return ""
	}
	// Zoned timestamps → UTC MySQL; bare wall times keep the clock as written
	// (matches WP date_query against post_date / post_modified).
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format("2006-01-02 15:04:05")
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return ""
}

// parsePostsSlug reads REST ?slug= (maps to post_name__in, P4.89).
// Accepts comma-separated slugs and/or repeated query values.
func parsePostsSlug(r *http.Request) []string {
	if r == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	q := r.URL.Query()
	if vals, ok := q["slug"]; ok {
		for _, v := range vals {
			add(v)
		}
	} else {
		add(q.Get("slug"))
	}
	return out
}

// parsePostsStatus reads REST ?status= (maps to WP_Query post_status, P4.88).
// Default is publish. Non-publish values require edit_posts (editors).
// Returns http status code + message on forbidden (0 on success).
func parsePostsStatus(r *http.Request, c *wp.Context) (statuses []string, code int, message string) {
	if r == nil {
		return []string{"publish"}, 0, ""
	}
	var raw []string
	for _, v := range r.URL.Query()["status"] {
		for _, part := range strings.Split(v, ",") {
			part = strings.ToLower(strings.TrimSpace(part))
			if part != "" {
				raw = append(raw, part)
			}
		}
	}
	if len(raw) == 0 {
		return []string{"publish"}, 0, ""
	}
	// Dedupe while preserving order.
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, s := range out {
		if s == "publish" {
			continue
		}
		if !canEditPosts(c) {
			code := http.StatusUnauthorized
			if c != nil && c.CurrentUser != nil {
				code = http.StatusForbidden
			}
			return nil, code, "Status is forbidden."
		}
	}
	return out, 0, ""
}

func parsePostsTermIDs(r *http.Request, key string) []int64 {
	if r == nil || key == "" {
		return nil
	}
	seen := map[int64]bool{}
	var out []int64
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.ParseInt(part, 10, 64)
			if err != nil || n <= 0 || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	q := r.URL.Query()
	if vals, ok := q[key]; ok {
		for _, v := range vals {
			add(v)
		}
	} else {
		add(q.Get(key))
	}
	return out
}

func countPosts(c *wp.Context, postType string, stickyFilter *bool, search string, author int64, categoryIn, tagIn []int64, statuses []string, slugs []string, include, exclude, parentIn, parentNotIn []int64, dateAfter, dateBefore, modifiedAfter, modifiedBefore string, formatIn []string, formatStandard bool) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	if postType == "" {
		postType = "post"
	}
	args := wp.QueryArgs{
		PostType:           postType,
		PostStatus:         "publish",
		PostStatuses:       statuses,
		Search:             search,
		Author:             author,
		CategoryIn:         categoryIn,
		TagIn:              tagIn,
		NameIn:             slugs,
		PostIn:             include,
		PostNotIn:          exclude,
		ParentIn:           parentIn,
		ParentNotIn:        parentNotIn,
		DateAfter:          dateAfter,
		DateBefore:         dateBefore,
		ModifiedAfter:      modifiedAfter,
		ModifiedBefore:     modifiedBefore,
		PostFormatIn:       formatIn,
		PostFormatStandard: formatStandard,
	}
	applyStickyQueryFilter(c, postType, stickyFilter, &args)
	return wp.CountPosts(c, args)
}

// applyStickyQueryFilter maps REST ?sticky=true|false onto post__in / post__not_in (P4.82).
// When include/exclude are also set, sticky=true intersects post__in and sticky=false
// merges into post__not_in (WP REST posts controller).
func applyStickyQueryFilter(c *wp.Context, postType string, stickyFilter *bool, args *wp.QueryArgs) {
	if stickyFilter == nil || args == nil || postType != "post" {
		return
	}
	ids := stickyIDList(c)
	if *stickyFilter {
		// Empty sticky set → no results (WP uses post__in = [0]).
		if len(args.PostIn) > 0 {
			args.PostIn = intersectIDs(ids, args.PostIn)
		} else {
			args.PostIn = ids
		}
		if args.PostIn == nil {
			args.PostIn = []int64{}
		}
		return
	}
	if len(ids) > 0 {
		args.PostNotIn = append(args.PostNotIn, ids...)
	}
}

func intersectIDs(a, b []int64) []int64 {
	if len(a) == 0 || len(b) == 0 {
		return []int64{}
	}
	set := map[int64]bool{}
	for _, id := range a {
		if id > 0 {
			set[id] = true
		}
	}
	out := make([]int64, 0, len(b))
	seen := map[int64]bool{}
	for _, id := range b {
		if id > 0 && set[id] && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func stickyIDList(c *wp.Context) []int64 {
	m := stickyPostIDs(c)
	if len(m) == 0 {
		return nil
	}
	out := make([]int64, 0, len(m))
	for id := range m {
		if id > 0 {
			out = append(out, id)
		}
	}
	return out
}

// parseStickyQuery reads REST ?sticky= as an optional boolean.
func parseStickyQuery(r *http.Request) *bool {
	if r == nil {
		return nil
	}
	raw := strings.TrimSpace(r.URL.Query().Get("sticky"))
	if raw == "" {
		return nil
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		v := true
		return &v
	case "0", "false", "no", "off":
		v := false
		return &v
	default:
		return nil
	}
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
	rendered := renderPostContentCtx(c, row.Content)
	cats := postTermIDs(c, row.ID, "category")
	tags := postTermIDs(c, row.ID, "post_tag")
	format := postFormat(c, row.ID, row.Type)
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
		Link:        itemPermalink(c, home, row),
		Title:       RenderedString{Rendered: row.Title},
		Content: ProtectedRendered{
			Rendered:  rendered,
			Protected: protected,
		},
		Excerpt:       buildExcerpt(row.Excerpt, rendered, protected),
		Author:        row.Author,
		FeaturedMedia: featuredMediaID(c, row.ID),
		CommentStatus: row.CommentStatus,
		PingStatus:    row.PingStatus,
		Sticky:        sticky[row.ID],
		Template:      pageTemplate(c, row.ID),
		Format:        format,
		Meta:          map[string]any{"footnotes": ""},
		Categories:    cats,
		Tags:          tags,
		Parent:        row.Parent,
		MenuOrder:     row.MenuOrder,
		ClassList:     buildClassList(c, row, format, cats),
		Links:         buildPostLinks(c, home, row, edit),
	}
	if edit {
		titleRaw, contentRaw, excerptRaw, guidRaw := row.Title, row.Content, row.Excerpt, row.GUID
		pass := row.Password
		p.Title.Raw = &titleRaw
		p.Content.Raw = &contentRaw
		p.Excerpt.Raw = &excerptRaw
		p.GUID.Raw = &guidRaw
		p.Password = &pass
		p.GeneratedSlug = wp.SanitizeTitle(row.Title)
		p.PermalinkTemplate = permalinkTemplate(home, row.Type, row.Date)
		bv := 0
		if strings.Contains(row.Content, "<!-- wp:") {
			bv = 1
		}
		p.Content.BlockVersion = &bv
	}
	return p
}

func buildClassList(c *wp.Context, row wp.Post, format string, catIDs []int64) []string {
	idStr := strconv.FormatInt(row.ID, 10)
	out := []string{"post-" + idStr}
	if row.Type == "page" {
		out = append(out, "page", "type-page", "status-"+row.Status, "hentry")
		return out
	}
	out = append(out, "post", "type-post", "status-"+row.Status)
	if format == "" {
		format = "standard"
	}
	out = append(out, "format-"+format, "hentry")
	for _, slug := range postTermSlugs(c, row.ID, "category") {
		out = append(out, "category-"+slug)
	}
	_ = catIDs
	return out
}

func buildPostLinks(c *wp.Context, home string, row wp.Post, edit bool) map[string]any {
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	base := "posts"
	if row.Type == "page" {
		base = "pages"
	}
	allow := []string{"GET"}
	if edit {
		allow = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	}
	revCount := 0
	if row.Type == "post" {
		revCount = countPostRevisions(c, row.ID)
	}
	links := map[string]any{
		"self": []map[string]any{{
			"href":        fmt.Sprintf("%s/%s/%d", restRoot, base, row.ID),
			"targetHints": map[string]any{"allow": allow},
		}},
		"collection": []map[string]any{{"href": restRoot + "/" + base}},
		"about":      []map[string]any{{"href": restRoot + "/types/" + row.Type}},
		"author": []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/users/%d", restRoot, row.Author),
		}},
		"replies": []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/comments?post=%d", restRoot, row.ID),
		}},
		"version-history": []map[string]any{{
			"count": revCount,
			"href":  fmt.Sprintf("%s/%s/%d/revisions", restRoot, base, row.ID),
		}},
		"wp:attachment": []map[string]any{{
			"href": fmt.Sprintf("%s/media?parent=%d", restRoot, row.ID),
		}},
		"curies": []map[string]any{{
			"name":      "wp",
			"href":      "https://api.w.org/{rel}",
			"templated": true,
		}},
	}
	if row.Type == "post" {
		links["wp:term"] = []map[string]any{
			{
				"taxonomy":   "category",
				"embeddable": true,
				"href":       fmt.Sprintf("%s/categories?post=%d", restRoot, row.ID),
			},
			{
				"taxonomy":   "post_tag",
				"embeddable": true,
				"href":       fmt.Sprintf("%s/tags?post=%d", restRoot, row.ID),
			},
		}
	}
	if row.Type == "page" && row.Parent > 0 {
		links["up"] = []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/pages/%d", restRoot, row.Parent),
		}}
	}
	if thumb := featuredMediaID(c, row.ID); thumb > 0 {
		links["wp:featuredmedia"] = []map[string]any{{
			"embeddable": true,
			"href":       fmt.Sprintf("%s/media/%d", restRoot, thumb),
		}}
	}
	if edit {
		selfHref := fmt.Sprintf("%s/%s/%d", restRoot, base, row.ID)
		for _, action := range []string{
			"wp:action-publish",
			"wp:action-unfiltered-html",
			"wp:action-sticky",
			"wp:action-assign-author",
			"wp:action-create-categories",
			"wp:action-assign-categories",
			"wp:action-create-tags",
			"wp:action-assign-tags",
		} {
			if row.Type == "page" && (action == "wp:action-sticky" ||
				strings.Contains(action, "categories") || strings.Contains(action, "tags")) {
				continue
			}
			links[action] = []map[string]any{{"href": selfHref}}
		}
	}
	return links
}

func permalinkTemplate(home, postType, mysqlDate string) string {
	home = strings.TrimRight(home, "/")
	if postType == "page" {
		return home + "/%pagename%/"
	}
	t, err := time.Parse("2006-01-02 15:04:05", mysqlDate)
	if err != nil {
		return home + "/%year%/%monthnum%/%day%/%postname%/"
	}
	return fmt.Sprintf("%s/%04d/%02d/%02d/%%postname%%/", home, t.Year(), int(t.Month()), t.Day())
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
	return wp.GetPostThumbnailID(c, postID)
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

func itemPermalink(c *wp.Context, home string, row wp.Post) string {
	if row.Type == "page" {
		return pagePermalink(c, home, row)
	}
	return postPermalink(home, row.Date, row.Name, row.ID)
}

func pagePermalink(c *wp.Context, home string, row wp.Post) string {
	if row.Name == "" {
		return fmt.Sprintf("%s/?page_id=%d", home, row.ID)
	}
	parts := []string{row.Name}
	parent := row.Parent
	seen := map[int64]bool{row.ID: true}
	for parent > 0 && !seen[parent] {
		seen[parent] = true
		p, err := wp.GetPost(c, wp.QueryArgs{P: parent, PostType: "page", PostStatus: "publish"})
		if err != nil || p == nil {
			break
		}
		if p.Name != "" {
			parts = append([]string{p.Name}, parts...)
		}
		parent = p.Parent
	}
	return fmt.Sprintf("%s/%s/", home, strings.Join(parts, "/"))
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
