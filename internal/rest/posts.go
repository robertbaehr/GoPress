package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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
	// Embedded is populated when ?_embed is set (P4.146).
	Embedded map[string]any `json:"_embedded,omitempty"`
	// Edit-context fields (nil/absent in view; pointer so "" is emitted when set).
	Password          *string `json:"password,omitempty"`
	PermalinkTemplate string  `json:"permalink_template,omitempty"`
	GeneratedSlug     string  `json:"generated_slug,omitempty"`
	// RESTContext is view|embed|edit; drives filter_response_by_context (P4.145).
	RESTContext string `json:"-"`
}

// MarshalJSON omits post-only fields on pages and page-only fields on posts.
// Embed context keeps the WP schema subset (P4.145).
func (p Post) MarshalJSON() ([]byte, error) {
	if p.RESTContext == "embed" {
		type embedOut struct {
			ID            int64             `json:"id"`
			Date          string            `json:"date"`
			Slug          string            `json:"slug"`
			Type          string            `json:"type"`
			Link          string            `json:"link"`
			Title         RenderedString    `json:"title"`
			Excerpt       ProtectedRendered `json:"excerpt"`
			Author        int64             `json:"author"`
			FeaturedMedia int64             `json:"featured_media"`
			Links         map[string]any    `json:"_links"`
			Embedded      map[string]any    `json:"_embedded,omitempty"`
		}
		return json.Marshal(embedOut{
			ID: p.ID, Date: p.Date, Slug: p.Slug, Type: p.Type, Link: p.Link,
			Title: p.Title, Excerpt: p.Excerpt, Author: p.Author,
			FeaturedMedia: p.FeaturedMedia, Links: p.Links, Embedded: p.Embedded,
		})
	}
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
			Embedded          map[string]any    `json:"_embedded,omitempty"`
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
			Embedded: p.Embedded,
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
		Embedded          map[string]any    `json:"_embedded,omitempty"`
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
		ClassList: p.ClassList, Links: p.Links, Embedded: p.Embedded,
		Password: p.Password, PermalinkTemplate: p.PermalinkTemplate,
		GeneratedSlug: p.GeneratedSlug,
	})
}

// parsePostsRESTContext returns view|embed|edit (default view). ok=false when invalid (P4.145).
func parsePostsRESTContext(r *http.Request) (string, bool) {
	ctx := strings.TrimSpace(r.URL.Query().Get("context"))
	if ctx == "" {
		return "view", true
	}
	switch ctx {
	case "view", "embed", "edit":
		return ctx, true
	default:
		return "", false
	}
}

// writeRESTInvalidPostsContext emits rest_invalid_param for bad context (P4.145).
func writeRESTInvalidPostsContext(w http.ResponseWriter) {
	msg := "context is not one of view, embed, and edit."
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"code":    "rest_invalid_param",
		"message": "Invalid parameter(s): context",
		"data": map[string]any{
			"status": 400,
			"params": map[string]any{
				"context": msg,
			},
			"details": map[string]any{
				"context": map[string]any{
					"code":    "rest_not_in_enum",
					"message": msg,
					"data":    nil,
				},
			},
		},
	})
}

// writeRESTForbiddenPostsContext emits rest_forbidden_context (P4.145).
// WP omits Allow on this 401/403 (unlike invalid-param responses).
func writeRESTForbiddenPostsContext(w http.ResponseWriter, c *wp.Context, item bool) {
	status := http.StatusUnauthorized
	if c != nil && c.CurrentUser != nil {
		status = http.StatusForbidden
	}
	msg := "Sorry, you are not allowed to edit posts in this post type."
	if item {
		msg = "Sorry, you are not allowed to edit this post."
	}
	writeJSON(w, status, map[string]any{
		"code":    "rest_forbidden_context",
		"message": msg,
		"data":    map[string]any{"status": status},
	})
}

func applyPostsRESTContext(posts []Post, ctx string) {
	for i := range posts {
		posts[i].RESTContext = ctx
	}
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

// parsePostsPerPage validates REST collection per_page (P4.138).
// Mirrors WP_REST_Controller::get_collection_params: integer, min 1, max 100.
// Absent → def; invalid/out-of-range → ok=false (caller returns rest_invalid_param).
func parsePostsPerPage(r *http.Request, def int) (perPage int, ok bool) {
	if def <= 0 {
		def = 10
	}
	if r == nil {
		return def, true
	}
	v := strings.TrimSpace(r.URL.Query().Get("per_page"))
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 100 {
		return 0, false
	}
	return n, true
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

// parsePostsOffset validates REST ?offset= (P4.92 / P4.139).
// Absent → useOffset=false, valid=true.
// Present but non-integer (incl. empty/float) → valid=false (rest_invalid_param).
// Negative integers are accepted (valid) but clamped to 0 for WP_Query.
func parsePostsOffset(r *http.Request) (offset int, useOffset bool, valid bool) {
	if r == nil {
		return 0, false, true
	}
	q := r.URL.Query()
	if !q.Has("offset") {
		return 0, false, true
	}
	v := strings.TrimSpace(q.Get("offset"))
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false, false
	}
	if n < 0 {
		n = 0
	}
	return n, true, true
}

func listPostsPaged(c *wp.Context, postType string, limit, page int, edit bool, stickyFilter *bool, ignoreSticky bool, orderby, order, search string, searchColumns []string, authorIn, authorNotIn []int64, categoryIn, categoryNotIn, tagIn, tagNotIn []int64, taxRelation string, statuses []string, slugs []string, include, exclude, parentIn, parentNotIn []int64, dateAfter, dateBefore, modifiedAfter, modifiedBefore string, formatIn []string, formatStandard bool, offsetOverride int, useOffset bool, menuOrderFilter *int) ([]Post, error) {
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
		PostType:           postType,
		PostStatus:         "publish",
		PostStatuses:       statuses,
		PostsPerPage:       limit,
		Offset:             offset,
		OrderBy:            orderby,
		Order:              order,
		Search:             search,
		SearchColumns:      searchColumns,
		AuthorIn:           authorIn,
		AuthorNotIn:        authorNotIn,
		CategoryIn:         categoryIn,
		CategoryNotIn:      categoryNotIn,
		TagIn:              tagIn,
		TagNotIn:           tagNotIn,
		TaxRelation:        taxRelation,
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
		MenuOrderFilter:    menuOrderFilter,
	}
	applyStickyQueryFilter(c, postType, stickyFilter, &args)
	// P4.141: ignore_sticky=false → StickyFirst (prepend stickies). When sticky=
	// sets post__in, WP unsets ignore_sticky_posts — skip StickyFirst then.
	if postType == "post" && !ignoreSticky && stickyFilter == nil && len(args.PostIn) == 0 {
		args.StickyFirst = true
	}
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

// parsePostsOrderbyTyped accepts REST orderby aliases (P4.83 / P4.94 / P4.131 / P4.149 / P4.159).
// Absent → ("date", true, false). Present but not in the post-type enum → ("", false, false).
// typeErr is true when orderby[]= is present (rest_invalid_type / not string).
func parsePostsOrderbyTyped(r *http.Request, postType string) (val string, ok bool, typeErr bool) {
	if r == nil {
		return "date", true, false
	}
	q := r.URL.Query()
	// P4.159: schema type string — reject orderby[]= (rest_invalid_type).
	if _, hasArray := q["orderby[]"]; hasArray {
		return "", false, true
	}
	if !q.Has("orderby") {
		return "date", true, false
	}
	v := strings.ToLower(strings.TrimSpace(q.Get("orderby")))
	switch v {
	case "author", "date", "id", "include", "modified", "parent",
		"relevance", "slug", "include_slugs", "title":
		return v, true, false
	case "menu_order":
		// P4.94 / P4.149: menu_order enum only for pages (page-attributes).
		if postType == "page" {
			return v, true, false
		}
		return "", false, false
	default:
		return "", false, false
	}
}

// parsePostsOrderTyped accepts REST order=asc|desc (P4.83 / P4.149 / P4.159).
// Absent → ("desc", true, false). Present but not asc|desc → ("", false, false).
// typeErr is true when order[]= is present (rest_invalid_type / not string).
func parsePostsOrderTyped(r *http.Request) (val string, ok bool, typeErr bool) {
	if r == nil {
		return "desc", true, false
	}
	q := r.URL.Query()
	// P4.159: schema type string — reject order[]= (rest_invalid_type).
	if _, hasArray := q["order[]"]; hasArray {
		return "", false, true
	}
	if !q.Has("order") {
		return "desc", true, false
	}
	v := strings.ToLower(strings.TrimSpace(q.Get("order")))
	if v == "asc" || v == "desc" {
		return v, true, false
	}
	return "", false, false
}

// writeRESTInvalidStringParams emits rest_invalid_param for non-string query args (P4.159).
func writeRESTInvalidStringParams(w http.ResponseWriter, keys []string) {
	params := map[string]any{}
	details := map[string]any{}
	for _, k := range keys {
		msg := k + " is not of type string."
		params[k] = msg
		details[k] = map[string]any{
			"code":    "rest_invalid_type",
			"message": msg,
			"data":    map[string]any{"param": k},
		}
	}
	msg := "Invalid parameter(s): " + strings.Join(keys, ", ")
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"code":    "rest_invalid_param",
		"message": msg,
		"data": map[string]any{
			"status":  400,
			"params":  params,
			"details": details,
		},
	})
}

// parsePostsMenuOrder reads REST ?menu_order= for pages (P4.149 / P4.157).
// Absent → use=false, ok=true. Present non-integer / empty / array form → ok=false.
// P4.157: reject menu_order[]= (integer, not array) and empty menu_order=.
func parsePostsMenuOrder(r *http.Request) (val int, use bool, ok bool) {
	if r == nil {
		return 0, false, true
	}
	q := r.URL.Query()
	// P4.157: array form is never valid for an integer collection arg.
	if _, hasArray := q["menu_order[]"]; hasArray {
		return 0, false, false
	}
	if !q.Has("menu_order") {
		return 0, false, true
	}
	v := strings.TrimSpace(q.Get("menu_order"))
	if v == "" {
		return 0, false, false
	}
	// P4.157: require a pure integer token (reject 1.5 / +1 / 1e2).
	for i, ch := range v {
		if ch >= '0' && ch <= '9' {
			continue
		}
		if ch == '-' && i == 0 {
			continue
		}
		return 0, false, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false, false
	}
	return n, true, true
}

// writeRESTInvalidOrderParams emits rest_invalid_param for bad order/orderby (P4.149).
// params maps query key → human message (rest_not_in_enum).
func writeRESTInvalidOrderParams(w http.ResponseWriter, params map[string]string) {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	details := map[string]any{}
	paramMsgs := map[string]any{}
	for _, k := range keys {
		msg := params[k]
		paramMsgs[k] = msg
		details[k] = map[string]any{
			"code":    "rest_not_in_enum",
			"message": msg,
			"data":    nil,
		}
	}
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"code":    "rest_invalid_param",
		"message": "Invalid parameter(s): " + strings.Join(keys, ", "),
		"data": map[string]any{
			"status":  400,
			"params":  paramMsgs,
			"details": details,
		},
	})
}

// postsOrderbyEnumMessage is the WP rest_not_in_enum message for orderby (P4.149).
func postsOrderbyEnumMessage(postType string) string {
	if postType == "page" {
		return "orderby is not one of author, date, id, include, modified, parent, relevance, slug, include_slugs, title, and menu_order."
	}
	return "orderby is not one of author, date, id, include, modified, parent, relevance, slug, include_slugs, and title."
}

// parsePostsSearch reads REST ?search= (maps to WP_Query s / QueryArgs.Search, P4.84).
func parsePostsSearch(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get("search"))
}

// parsePostsSearchColumns reads REST ?search_columns= (P4.135 / P4.153).
// Accepts comma-separated values and/or repeated query keys.
// P4.153: invalid enum tokens → errMsg for rest_invalid_param / rest_not_in_enum
// (WP schema items.enum). Empty tokens are skipped before indexing (WP).
func parsePostsSearchColumns(r *http.Request) (cols []string, errMsg string) {
	if r == nil {
		return nil, ""
	}
	allowed := map[string]bool{
		"post_title":   true,
		"post_content": true,
		"post_excerpt": true,
	}
	var tokens []string
	// keepEmpty: search_columns[]= contributes an empty enum token (WP 400);
	// scalar search_columns= / comma gaps skip empties before indexing.
	add := func(raw string, keepEmpty bool) {
		if keepEmpty && raw == "" {
			tokens = append(tokens, "")
			return
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			tokens = append(tokens, part)
		}
	}
	q := r.URL.Query()
	for _, v := range q["search_columns"] {
		add(v, false)
	}
	for _, v := range q["search_columns[]"] {
		add(v, true)
	}
	var out []string
	seen := map[string]bool{}
	for i, part := range tokens {
		if !allowed[part] {
			return nil, "search_columns[" + strconv.Itoa(i) + "] is not one of post_title, post_content, and post_excerpt."
		}
		if seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out, ""
}

// parsePostsAuthor reads REST ?author= (maps to WP_Query author__in, P4.85 / P4.132).
// Accepts comma-separated IDs and/or repeated query values.
func parsePostsAuthor(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "author")
}

// parsePostsAuthorExclude reads REST ?author_exclude= (author__not_in, P4.132 / P4.152).
// Present but non-integer → ok=false (rest_invalid_param / rest_invalid_type).
func parsePostsAuthorExclude(r *http.Request) ([]int64, bool) {
	return parsePostsAuthorIDs(r, "author_exclude")
}

// parsePostsAuthorIDs parses author / author_exclude with strict integer validation (P4.152).
// Returns (nil, true) when absent or scalar-empty. Supports key=1,2, repeated ?key=, and key[]=.
// Present but non-integer tokens → (nil, false). Positive IDs only (author__in/not_in).
func parsePostsAuthorIDs(r *http.Request, key string) ([]int64, bool) {
	if r == nil || key == "" {
		return nil, true
	}
	q := r.URL.Query()
	scalarVals, hasScalar := q[key]
	arrayVals, hasArray := q[key+"[]"]
	if !hasScalar && !hasArray {
		return nil, true
	}
	seen := map[int64]bool{}
	var out []int64
	anyToken := false
	addCSV := func(raw string, arrayElem bool) bool {
		// PHP ?author_exclude[]= yields ['']; empty array elements fail integer validation.
		if arrayElem && strings.TrimSpace(raw) == "" {
			return false
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return true
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			anyToken = true
			n, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return false
			}
			// author__in / author__not_in ignore non-positive IDs (WP_Query).
			if n <= 0 || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
		return true
	}
	for _, v := range scalarVals {
		if !addCSV(v, false) {
			return nil, false
		}
	}
	for _, v := range arrayVals {
		if !addCSV(v, true) {
			return nil, false
		}
	}
	if !anyToken {
		return nil, true
	}
	return out, true
}

// parsePostsCategories reads REST ?categories= (maps to tax_query category term_id IN, P4.86).
// Accepts comma-separated IDs and/or repeated query values.
func parsePostsCategories(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "categories")
}

// parsePostsCategoriesExclude reads REST ?categories_exclude= (tax_query NOT IN, P4.136 / P4.155).
// Present but non-integer → ok=false (rest_invalid_param / rest_invalid_type).
func parsePostsCategoriesExclude(r *http.Request) ([]int64, bool) {
	return parsePostsStrictTermIDs(r, "categories_exclude")
}

// parsePostsStrictTermIDs parses term ID query args with strict integer validation (P4.155).
// Returns (nil, true) when absent or scalar-empty. Supports key=1,2, repeated ?key=, and key[]=.
// Present but non-integer tokens → (nil, false). Positive IDs only.
func parsePostsStrictTermIDs(r *http.Request, key string) ([]int64, bool) {
	if r == nil || key == "" {
		return nil, true
	}
	q := r.URL.Query()
	scalarVals, hasScalar := q[key]
	arrayVals, hasArray := q[key+"[]"]
	if !hasScalar && !hasArray {
		return nil, true
	}
	seen := map[int64]bool{}
	var out []int64
	anyToken := false
	addCSV := func(raw string, arrayElem bool) bool {
		// PHP ?categories_exclude[]= yields ['']; empty array elements fail integer validation.
		if arrayElem && strings.TrimSpace(raw) == "" {
			return false
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return true
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			anyToken = true
			n, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return false
			}
			if n <= 0 || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
		return true
	}
	for _, v := range scalarVals {
		if !addCSV(v, false) {
			return nil, false
		}
	}
	for _, v := range arrayVals {
		if !addCSV(v, true) {
			return nil, false
		}
	}
	if !anyToken {
		return nil, true
	}
	return out, true
}

// parsePostsTags reads REST ?tags= (maps to tax_query post_tag term_id IN, P4.87).
// Accepts comma-separated IDs and/or repeated query values.
func parsePostsTags(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "tags")
}

// parsePostsTagsExclude reads REST ?tags_exclude= (tax_query NOT IN, P4.136 / P4.156).
// Present but non-integer → ok=false (rest_invalid_param / rest_invalid_type).
func parsePostsTagsExclude(r *http.Request) ([]int64, bool) {
	return parsePostsStrictTermIDs(r, "tags_exclude")
}

// parsePostsTaxRelation reads REST ?tax_relation=AND|OR (P4.136). Empty → AND.
func parsePostsTaxRelation(r *http.Request) string {
	if r == nil {
		return ""
	}
	v := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("tax_relation")))
	if v == "OR" {
		return "OR"
	}
	if v == "AND" {
		return "AND"
	}
	return ""
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

// postsFormatEnumMessage is the WP rest_not_in_enum message for format[i] (P4.154).
func postsFormatEnumMessage(i int) string {
	return "format[" + strconv.Itoa(i) + "] is not one of standard, aside, chat, gallery, link, image, quote, status, video, and audio."
}

// parsePostsFormat reads REST ?format= (post_format tax_query, P4.96 / P4.154).
// Accepts comma-separated values and/or format[]= array keys.
// P4.154: invalid enum tokens → errMsg for rest_invalid_param / rest_not_in_enum
// (WP schema items.enum; case-sensitive). Empty scalar tokens are skipped before
// indexing; format[]= empty contributes an empty enum token (WP 400).
// Returns post-format-* slugs plus whether standard (NOT EXISTS) is requested.
func parsePostsFormat(r *http.Request) (formatIn []string, includeStandard bool, errMsg string) {
	if r == nil {
		return nil, false, ""
	}
	q := r.URL.Query()
	scalarVals, hasScalar := q["format"]
	arrayVals, hasArray := q["format[]"]
	if !hasScalar && !hasArray {
		return nil, false, ""
	}
	var tokens []string
	add := func(raw string, keepEmpty bool) {
		if keepEmpty && raw == "" {
			tokens = append(tokens, "")
			return
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			tokens = append(tokens, part)
		}
	}
	for _, v := range scalarVals {
		add(v, false)
	}
	for _, v := range arrayVals {
		add(v, true)
	}
	seen := map[string]bool{}
	for i, raw := range tokens {
		if !knownPostFormats[raw] {
			return nil, false, postsFormatEnumMessage(i)
		}
		if seen[raw] {
			continue
		}
		seen[raw] = true
		if raw == "standard" {
			includeStandard = true
			continue
		}
		formatIn = append(formatIn, "post-format-"+raw)
	}
	return formatIn, includeStandard, ""
}

// parsePostsInclude reads REST ?include= / include[]= (maps to post__in, P4.90 / P4.137).
func parsePostsInclude(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "include")
}

// parsePostsExclude reads REST ?exclude= / exclude[]= (maps to post__not_in, P4.90 / P4.137).
func parsePostsExclude(r *http.Request) []int64 {
	return parsePostsTermIDs(r, "exclude")
}

// parsePostsParent reads REST ?parent= (maps to post_parent__in, P4.93 / P4.142).
// Allows 0 for top-level hierarchical posts/pages. nil when absent or empty.
// Present but non-integer → ok=false (rest_invalid_param / rest_invalid_type).
func parsePostsParent(r *http.Request) ([]int64, bool) {
	return parsePostsParentIDs(r, "parent")
}

// parsePostsParentExclude reads REST ?parent_exclude= (post_parent__not_in, P4.93 / P4.142).
func parsePostsParentExclude(r *http.Request) ([]int64, bool) {
	return parsePostsParentIDs(r, "parent_exclude")
}

// parsePostsParentIDs parses parent / parent_exclude, allowing 0 and negatives.
// Returns (nil, true) when the query key is absent or scalar-empty (no filter).
// Supports key=1,2, repeated ?key=, and key[]=1&key[]=2 (P4.142).
// Present but non-integer tokens → (nil, false).
func parsePostsParentIDs(r *http.Request, key string) ([]int64, bool) {
	if r == nil || key == "" {
		return nil, true
	}
	q := r.URL.Query()
	scalarVals, hasScalar := q[key]
	arrayVals, hasArray := q[key+"[]"]
	if !hasScalar && !hasArray {
		return nil, true
	}
	seen := map[int64]bool{}
	var out []int64
	// Track whether any non-empty token was seen (scalar "" alone → no filter).
	anyToken := false
	addCSV := func(raw string, arrayElem bool) bool {
		// PHP ?parent[]= yields ['']; empty array elements fail integer validation.
		if arrayElem && strings.TrimSpace(raw) == "" {
			return false
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return true
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			anyToken = true
			n, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return false
			}
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
		return true
	}
	for _, v := range scalarVals {
		if !addCSV(v, false) {
			return nil, false
		}
	}
	for _, v := range arrayVals {
		if !addCSV(v, true) {
			return nil, false
		}
	}
	if !anyToken {
		// parent= / parent_exclude= → empty list after sanitize → no filter (WP).
		return nil, true
	}
	return out, true
}

// writeRESTInvalidIntegerArrayParams emits rest_invalid_param for bad integer-array query args (P4.142).
func writeRESTInvalidIntegerArrayParams(w http.ResponseWriter, keys []string) {
	params := map[string]any{}
	details := map[string]any{}
	for _, k := range keys {
		msg := k + "[0] is not of type integer."
		params[k] = msg
		details[k] = map[string]any{
			"code":    "rest_invalid_type",
			"message": msg,
			"data":    map[string]any{"param": k + "[0]"},
		}
	}
	msg := "Invalid parameter(s): " + strings.Join(keys, ", ")
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"code":    "rest_invalid_param",
		"message": msg,
		"data": map[string]any{
			"status":  400,
			"params":  params,
			"details": details,
		},
	})
}

// parsePostsAfter reads REST ?after= (ISO8601 → MySQL datetime, P4.91 / P4.140).
// Present but invalid/empty → ok=false (rest_invalid_param / rest_invalid_date).
func parsePostsAfter(r *http.Request) (string, bool) {
	return parseRESTDateTimeQuery(r, "after")
}

// parsePostsBefore reads REST ?before= (ISO8601 → MySQL datetime, P4.91 / P4.140).
func parsePostsBefore(r *http.Request) (string, bool) {
	return parseRESTDateTimeQuery(r, "before")
}

// parsePostsModifiedAfter reads REST ?modified_after= (P4.95 / P4.140).
func parsePostsModifiedAfter(r *http.Request) (string, bool) {
	return parseRESTDateTimeQuery(r, "modified_after")
}

// parsePostsModifiedBefore reads REST ?modified_before= (P4.95 / P4.140).
func parsePostsModifiedBefore(r *http.Request) (string, bool) {
	return parseRESTDateTimeQuery(r, "modified_before")
}

// parseRESTDateTimeQuery parses an ISO8601 / MySQL datetime query arg to MySQL form.
// Absent → ("", true). Present but empty/unparseable → ("", false) (P4.140).
func parseRESTDateTimeQuery(r *http.Request, key string) (string, bool) {
	if r == nil || key == "" {
		return "", true
	}
	q := r.URL.Query()
	if !q.Has(key) {
		return "", true
	}
	raw := strings.TrimSpace(q.Get(key))
	if raw == "" {
		return "", false
	}
	// Zoned timestamps → UTC MySQL; bare wall times keep the clock as written
	// (matches WP date_query against post_date / post_modified).
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format("2006-01-02 15:04:05"), true
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return t.Format("2006-01-02 15:04:05"), true
		}
	}
	return "", false
}

// writeRESTInvalidDateParams emits rest_invalid_param for invalid date-time args (P4.140).
func writeRESTInvalidDateParams(w http.ResponseWriter, keys []string) {
	params := map[string]any{}
	details := map[string]any{}
	for _, k := range keys {
		params[k] = "Invalid date."
		details[k] = map[string]any{
			"code":    "rest_invalid_date",
			"message": "Invalid date.",
			"data":    nil,
		}
	}
	msg := "Invalid parameter(s): " + strings.Join(keys, ", ")
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"code":    "rest_invalid_param",
		"message": msg,
		"data": map[string]any{
			"status":  400,
			"params":  params,
			"details": details,
		},
	})
}

// parsePostsSlug reads REST ?slug= (maps to post_name__in, P4.89 / P4.133 / P4.143).
// Accepts comma-separated slugs, repeated ?slug=, PHP-style slug[]= arrays, and
// indexed slug[0]= / slug[1]= forms. Collection query params are not
// sanitize_title'd (WP REST slug query has no sanitize_callback).
//
// Scalar slug= (empty) → no filter. Present array form with only empty elements
// (slug[]= / slug[0]=) → empty NameIn filter (zero results), matching WP.
func parsePostsSlug(r *http.Request) []string {
	if r == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	arrayPresent := false
	addScalar := func(raw string) {
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
	addArray := func(raw string) {
		arrayPresent = true
		raw = strings.TrimSpace(raw)
		// Empty array element is retained as a filter token (→ post_name IN ('')).
		if raw == "" {
			if !seen[""] {
				seen[""] = true
				out = append(out, "")
			}
			return
		}
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				if !seen[""] {
					seen[""] = true
					out = append(out, "")
				}
				continue
			}
			if seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	q := r.URL.Query()
	if vals, ok := q["slug"]; ok {
		for _, v := range vals {
			addScalar(v)
		}
	}
	// PHP array form: slug[]=a&slug[]=b (rest_sanitize_array / WP_REST_Request).
	if vals, ok := q["slug[]"]; ok {
		for _, v := range vals {
			addArray(v)
		}
	}
	// Indexed form: slug[0]=a&slug[1]=b — collect in numeric index order (P4.143).
	type idxVal struct {
		idx int
		val string
	}
	var indexed []idxVal
	for key, vals := range q {
		if !strings.HasPrefix(key, "slug[") || !strings.HasSuffix(key, "]") || key == "slug[]" {
			continue
		}
		inner := key[len("slug[") : len(key)-1]
		n, err := strconv.Atoi(inner)
		if err != nil {
			// Non-numeric slug[foo]= — still treat as array element (WP keeps key order).
			for _, v := range vals {
				addArray(v)
			}
			continue
		}
		for _, v := range vals {
			indexed = append(indexed, idxVal{idx: n, val: v})
		}
	}
	if len(indexed) > 0 {
		sort.SliceStable(indexed, func(i, j int) bool { return indexed[i].idx < indexed[j].idx })
		for _, iv := range indexed {
			addArray(iv.val)
		}
	}
	if len(out) == 0 {
		if arrayPresent {
			return []string{""}
		}
		return nil
	}
	return out
}

// postsRESTStatusEnum is the WP posts collection status items.enum (P4.151).
var postsRESTStatusEnum = []string{
	"publish", "future", "draft", "pending", "private", "trash", "auto-draft",
	"inherit", "request-pending", "request-confirmed", "request-failed",
	"request-completed", "any",
}

var postsRESTStatusEnumSet = func() map[string]bool {
	m := make(map[string]bool, len(postsRESTStatusEnum))
	for _, s := range postsRESTStatusEnum {
		m[s] = true
	}
	return m
}()

// postsRESTStatusEnumMessage matches WP rest_not_in_enum for status[0] (P4.151).
const postsRESTStatusEnumMessage = "status[0] is not one of publish, future, draft, pending, private, trash, auto-draft, inherit, request-pending, request-confirmed, request-failed, request-completed, and any."

// parsePostsStatus reads REST ?status= (maps to WP_Query post_status, P4.88 / P4.134 / P4.151).
// Default is publish. Non-publish values (incl. "any") require edit_posts (editors).
// Accepts comma-separated values, repeated ?status=, and PHP-style status[]= arrays.
// Tokens are sanitize_key'd (lowercase a-z0-9_-).
// On forbidden: forbiddenCode is 401/403 (caller wraps as rest_invalid_param).
// On bad enum (editors): enumInvalid is true.
func parsePostsStatus(r *http.Request, c *wp.Context) (statuses []string, forbiddenCode int, enumInvalid bool) {
	if r == nil {
		return []string{"publish"}, 0, false
	}
	var raw []string
	add := func(v string) {
		for _, part := range strings.Split(v, ",") {
			part = sanitizeRESTStatusKey(part)
			if part != "" {
				raw = append(raw, part)
			}
		}
	}
	q := r.URL.Query()
	if vals, ok := q["status"]; ok {
		for _, v := range vals {
			add(v)
		}
	}
	// PHP array form: status[]=draft&status[]=pending (rest_sanitize_array).
	if vals, ok := q["status[]"]; ok {
		for _, v := range vals {
			add(v)
		}
	}
	if len(raw) == 0 {
		return []string{"publish"}, 0, false
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
		// P4.151 / sanitize_post_statuses: capability check before enum validate.
		if !canEditPosts(c) {
			code := http.StatusUnauthorized
			if c != nil && c.CurrentUser != nil {
				code = http.StatusForbidden
			}
			return nil, code, false
		}
		if !postsRESTStatusEnumSet[s] {
			return nil, 0, true
		}
	}
	return out, 0, false
}

// writeRESTInvalidPostsStatus emits rest_invalid_param for status query errors (P4.151).
// Mirrors WP: outer 400 rest_invalid_param wrapping rest_forbidden_status or rest_not_in_enum.
func writeRESTInvalidPostsStatus(w http.ResponseWriter, forbiddenCode int, enumInvalid bool) {
	w.Header().Set("Allow", "GET")
	if enumInvalid {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): status",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{
					"status": postsRESTStatusEnumMessage,
				},
				"details": map[string]any{
					"status": map[string]any{
						"code":    "rest_not_in_enum",
						"message": postsRESTStatusEnumMessage,
						"data":    nil,
					},
				},
			},
		})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"code":    "rest_invalid_param",
		"message": "Invalid parameter(s): status",
		"data": map[string]any{
			"status": 400,
			"params": map[string]any{
				"status": "Status is forbidden.",
			},
			"details": map[string]any{
				"status": map[string]any{
					"code":    "rest_forbidden_status",
					"message": "Status is forbidden.",
					"data":    map[string]any{"status": forbiddenCode},
				},
			},
		},
	})
}

// sanitizeRESTStatusKey mirrors sanitize_key for REST post status tokens (P4.134).
func sanitizeRESTStatusKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parsePostsTermIDs parses comma-separated / repeated / PHP-array ID query params
// (P4.90 / P4.137). Accepts key=1,2, repeated ?key=, and key[]=1&key[]=2.
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
	}
	// PHP array form: include[]=1&include[]=13 (rest_sanitize_array / WP_REST_Request).
	if vals, ok := q[key+"[]"]; ok {
		for _, v := range vals {
			add(v)
		}
	}
	if len(out) == 0 {
		add(q.Get(key))
	}
	return out
}

func countPosts(c *wp.Context, postType string, stickyFilter *bool, search string, searchColumns []string, authorIn, authorNotIn []int64, categoryIn, categoryNotIn, tagIn, tagNotIn []int64, taxRelation string, statuses []string, slugs []string, include, exclude, parentIn, parentNotIn []int64, dateAfter, dateBefore, modifiedAfter, modifiedBefore string, formatIn []string, formatStandard bool, menuOrderFilter *int) (int, error) {
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
		SearchColumns:      searchColumns,
		AuthorIn:           authorIn,
		AuthorNotIn:        authorNotIn,
		CategoryIn:         categoryIn,
		CategoryNotIn:      categoryNotIn,
		TagIn:              tagIn,
		TagNotIn:           tagNotIn,
		TaxRelation:        taxRelation,
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
		MenuOrderFilter:    menuOrderFilter,
	}
	applyStickyQueryFilter(c, postType, stickyFilter, &args)
	return wp.CountPosts(c, args)
}

// applyStickyQueryFilter maps REST ?sticky=true|false onto post__in / post__not_in (P4.82 / P4.158).
// When include/exclude are also set, sticky=true intersects post__in and sticky=false
// merges into post__not_in (WP REST posts controller).
// P4.158: empty sticky_posts (or empty intersection) → post__in = [0].
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
		if len(args.PostIn) == 0 {
			args.PostIn = []int64{0}
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
	// P4.158: prefer shared StickyPostIDs (sticky_posts option parser).
	if ids := wp.StickyPostIDs(c); len(ids) > 0 {
		return ids
	}
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

// stickyField resolves the REST sticky property via is_sticky (P4.158).
// Falls back to the preloaded sticky_posts map when hooks are unavailable.
func stickyField(c *wp.Context, id int64, sticky map[int64]bool) bool {
	if c != nil && c.Hooks != nil {
		return wp.IsSticky(c, id)
	}
	return sticky[id]
}

// parseStickyQuery reads REST ?sticky= as an optional boolean (P4.82 / P2.120 / P4.141).
// Absent → (nil, true). Present but empty/non-boolean → (nil, false) → rest_invalid_param.
// Accepts rest_is_boolean values and sanitizes via rest_sanitize_boolean.
func parseStickyQuery(r *http.Request) (*bool, bool) {
	if r == nil {
		return nil, true
	}
	q := r.URL.Query()
	if !q.Has("sticky") {
		return nil, true
	}
	raw := strings.TrimSpace(q.Get("sticky"))
	if raw == "" || !wp.RestIsBoolean(raw) {
		return nil, false
	}
	v := wp.RestSanitizeBoolean(raw)
	return &v, true
}

// parseIgnoreStickyQuery reads REST ?ignore_sticky= (P4.141).
// Absent → default true (WP schema). Present but invalid → ok=false.
// When false, sticky posts are prepended (StickyFirst); when true, no prepend.
func parseIgnoreStickyQuery(r *http.Request) (ignore bool, ok bool) {
	if r == nil {
		return true, true
	}
	q := r.URL.Query()
	if !q.Has("ignore_sticky") {
		return true, true // WP default
	}
	raw := strings.TrimSpace(q.Get("ignore_sticky"))
	if raw == "" || !wp.RestIsBoolean(raw) {
		return true, false
	}
	return wp.RestSanitizeBoolean(raw), true
}

// writeRESTInvalidBooleanParams emits rest_invalid_param for bad boolean query args (P4.141).
func writeRESTInvalidBooleanParams(w http.ResponseWriter, keys []string) {
	params := map[string]any{}
	details := map[string]any{}
	for _, k := range keys {
		msg := k + " is not of type boolean."
		params[k] = msg
		details[k] = map[string]any{
			"code":    "rest_invalid_type",
			"message": msg,
			"data":    map[string]any{"param": k},
		}
	}
	msg := "Invalid parameter(s): " + strings.Join(keys, ", ")
	w.Header().Set("Allow", "GET")
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"code":    "rest_invalid_param",
		"message": msg,
		"data": map[string]any{
			"status":  400,
			"params":  params,
			"details": details,
		},
	})
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
		ModifiedGMT: modifiedGMTToREST(c, row.ModifiedGMT, row.Modified),
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
		// P4.158: sticky field via is_sticky (option + is_sticky filter).
		Sticky:     stickyField(c, row.ID, sticky),
		Template:   pageTemplate(c, row.ID),
		Format:     format,
		Meta:       postMetaFields(c, row.ID),
		Categories: cats,
		Tags:       tags,
		Parent:     row.Parent,
		MenuOrder:  row.MenuOrder,
		ClassList:  buildClassList(c, row, format, cats),
		Links:      buildPostLinks(c, home, row, edit),
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

// postMetaFields returns registered REST post meta (footnotes subset, P4.130).
func postMetaFields(c *wp.Context, postID int64) map[string]any {
	out := map[string]any{"footnotes": ""}
	if c == nil || c.Meta == nil || postID <= 0 {
		return out
	}
	if v := asString(c.Meta.GetMetadata(wp.MetaPost, postID, "footnotes", true)); v != "" {
		out["footnotes"] = v
	}
	return out
}

// applyPostMetaWrite updates registered post meta keys from a REST write body (P4.130).
// Only footnotes is supported; omit means leave unchanged; empty string clears.
func applyPostMetaWrite(c *wp.Context, postID int64, meta map[string]any) error {
	if c == nil || c.Meta == nil || postID <= 0 || meta == nil {
		return nil
	}
	if raw, ok := meta["footnotes"]; ok {
		val := asString(raw)
		if _, _, _, err := c.Meta.UpdateMetadata(wp.MetaPost, postID, "footnotes", val, ""); err != nil {
			return err
		}
	}
	return nil
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
	if s == "" || isZeroMySQLDate(s) {
		return ""
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return strings.ReplaceAll(s, " ", "T")
	}
	return t.Format("2006-01-02T15:04:05")
}

// isZeroMySQLDate reports WP's empty datetime sentinel.
func isZeroMySQLDate(s string) bool {
	s = strings.TrimSpace(s)
	return s == "0000-00-00 00:00:00" || s == "0000-00-00T00:00:00"
}

// modifiedGMTToREST mirrors WP REST prepare_item_for_response modified_gmt:
// when post_modified_gmt is the zero sentinel, shim from post_modified using
// the site timezone / gmt_offset (P4.144).
func modifiedGMTToREST(c *wp.Context, modifiedGMT, modified string) string {
	gmt := strings.TrimSpace(modifiedGMT)
	if isZeroMySQLDate(gmt) || gmt == "" {
		local := strings.TrimSpace(modified)
		if local == "" || isZeroMySQLDate(local) {
			return ""
		}
		converted, err := wp.GetGMTFromDate(local, wp.TimezoneString(c), "Y-m-d H:i:s")
		if err != nil || strings.TrimSpace(converted) == "" {
			// Fallback: gmt_offset hours (WP casts float→int).
			offset := 0.0
			if c != nil && c.Options != nil {
				raw := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("gmt_offset", "0")))
				if n, err := strconv.ParseFloat(raw, 64); err == nil {
					offset = n
				}
			}
			t, err := time.Parse("2006-01-02 15:04:05", local)
			if err != nil {
				return ""
			}
			t = t.Add(-time.Duration(int(offset)) * time.Hour)
			return t.Format("2006-01-02T15:04:05")
		}
		gmt = converted
	}
	return mysqlDateToREST(gmt)
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
