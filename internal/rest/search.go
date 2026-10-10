package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/db"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// SearchResult is a wp/v2 search result object (view context).
// ID may be an int64 (posts/terms) or string (post-format slugs) per WP schema.
type SearchResult struct {
	ID      any            `json:"id"`
	Title   string         `json:"title"`
	URL     string         `json:"url"`
	Type    string         `json:"type"`
	Subtype string         `json:"subtype,omitempty"`
	Links   map[string]any `json:"_links"`
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
	searchType := strings.TrimSpace(r.URL.Query().Get("type"))
	if searchType == "" {
		searchType = "post"
	}
	switch searchType {
	case "post", "term", "post-format":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"code":    "rest_invalid_param",
			"message": "Invalid parameter(s): type",
			"data": map[string]any{
				"status": 400,
				"params": map[string]any{
					"type": "type is not one of post, term, and post-format.",
				},
				"details": map[string]any{
					"type": map[string]any{
						"code":    "rest_not_in_enum",
						"message": "type is not one of post, term, and post-format.",
						"data":    nil,
					},
				},
			},
		})
		return
	}

	perPage := parsePerPage(r, 10)
	page := parsePage(r)
	term := strings.TrimSpace(r.URL.Query().Get("search"))
	subtype := parseSearchSubtypes(r.URL.Query()["subtype"])
	// P4.100: include/exclude ID filters (WP_REST_Search_Controller collection params).
	include := parsePostsInclude(r)
	exclude := parsePostsExclude(r)
	// P4.122: collection orderby/order (parent REST controller params).
	orderby := parseSearchOrderBy(r, searchType)
	order := parseSearchOrder(r, searchType)

	var (
		total   int
		results []SearchResult
		err     error
	)
	switch searchType {
	case "term":
		total, err = countTermSearch(c, term, subtype, include, exclude)
		if err == nil {
			results, err = listTermSearchPaged(c, term, subtype, include, exclude, orderby, order, perPage, page)
		}
	case "post-format":
		total, results, err = listPostFormatSearch(c, term, perPage, page)
	default:
		postTypes := resolvePostSearchSubtypes(subtype)
		// P4.122: rest_post_search_query filter (query-arg map subset).
		postTypes, term, include, exclude, orderby, order = applyRestPostSearchQuery(c, r, postTypes, term, include, exclude, orderby, order)
		total, err = countSearch(c, term, postTypes, include, exclude)
		if err == nil {
			results, err = listSearchPaged(c, term, postTypes, include, exclude, orderby, order, perPage, page)
		}
	}
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

func parseSearchSubtypes(raw []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range raw {
		for _, s := range strings.Split(part, ",") {
			s = strings.TrimSpace(s)
			if s == "" || s == "any" {
				continue
			}
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out
}

// publicRESTPostSearchTypes are post subtypes exposed by WP_REST_Post_Search_Handler
// (public + show_in_rest, excluding attachment).
var publicRESTPostSearchTypes = []string{"post", "page"}

// resolvePostSearchSubtypes filters post search to requested subtypes (P4.67).
// Empty / unknown / "any" → all publicRESTPostSearchTypes.
func resolvePostSearchSubtypes(subtype []string) []string {
	if len(subtype) == 0 {
		return append([]string(nil), publicRESTPostSearchTypes...)
	}
	allowed := map[string]bool{}
	for _, t := range publicRESTPostSearchTypes {
		allowed[t] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, s := range subtype {
		if !allowed[s] || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 {
		return append([]string(nil), publicRESTPostSearchTypes...)
	}
	return out
}

func listSearchPaged(c *wp.Context, term string, postTypes []string, include, exclude []int64, orderby, order string, limit, page int) ([]SearchResult, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("nil DB")
	}
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	if len(postTypes) == 0 {
		postTypes = append([]string(nil), publicRESTPostSearchTypes...)
	}
	offset := (page - 1) * limit
	like := "%" + db.EscapeLike(term) + "%"
	args := make([]any, 0, len(postTypes)+len(include)+len(exclude)+6)
	args = append(args, "publish")
	inParts := make([]string, len(postTypes))
	for i, pt := range postTypes {
		inParts[i] = "%s"
		args = append(args, pt)
	}
	whereExtra, args := searchIncludeExcludeSQL("ID", include, exclude, args)
	args = append(args, like, like, like)
	orderSQL, orderArgs := searchPostOrderSQL(orderby, order, include)
	args = append(args, orderArgs...)
	args = append(args, limit, offset)
	q, prepArgs, err := c.DB.Prepare(
		"SELECT ID, post_date, post_name, post_title, post_type, post_parent FROM "+c.DB.Posts()+
			" WHERE post_status = %s AND post_type IN ("+strings.Join(inParts, ", ")+
			")"+whereExtra+" AND (post_title LIKE %s OR post_content LIKE %s OR post_excerpt LIKE %s) "+
			"ORDER BY "+orderSQL+" LIMIT %d OFFSET %d",
		args...,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, prepArgs...)
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]SearchResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, searchFromRow(c, row, home))
	}
	return out, nil
}

// parseSearchOrderBy reads REST ?orderby= for search collections (P4.122).
// Posts default date; terms default name.
func parseSearchOrderBy(r *http.Request, searchType string) string {
	def := "date"
	if searchType == "term" {
		def = "name"
	}
	if r == nil {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("orderby"))) {
	case "id":
		return "id"
	case "title", "name":
		if searchType == "term" {
			return "name"
		}
		return "title"
	case "slug":
		return "slug"
	case "include":
		return "include"
	case "relevance":
		return "relevance"
	case "date":
		return "date"
	default:
		return def
	}
}

// parseSearchOrder reads REST ?order= for search (P4.122).
// Posts default desc; terms default asc.
func parseSearchOrder(r *http.Request, searchType string) string {
	def := "desc"
	if searchType == "term" {
		def = "asc"
	}
	if r == nil {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order"))) {
	case "asc":
		return "asc"
	case "desc":
		return "desc"
	default:
		return def
	}
}

// searchPostOrderSQL builds ORDER BY for post search (P4.122).
func searchPostOrderSQL(orderby, order string, include []int64) (string, []any) {
	dir := "DESC"
	if order == "asc" {
		dir = "ASC"
	}
	switch orderby {
	case "id":
		return "ID " + dir, nil
	case "title":
		return "post_title " + dir + ", ID ASC", nil
	case "slug":
		return "post_name " + dir + ", ID ASC", nil
	case "include":
		if len(include) == 0 {
			return "post_date DESC, ID ASC", nil
		}
		parts := make([]string, len(include))
		args := make([]any, len(include))
		for i, id := range include {
			parts[i] = "%d"
			args[i] = id
		}
		return "FIELD(ID, " + strings.Join(parts, ", ") + ")", args
	case "relevance":
		// Without FULLTEXT, approximate relevance as title match priority then date.
		return "post_title ASC, post_date DESC, ID ASC", nil
	default: // date
		return "post_date " + dir + ", ID ASC", nil
	}
}

// applyRestPostSearchQuery applies rest_post_search_query (P4.122).
// Filter value is map[string]any with keys: s, post_type ([]string), post__in,
// post__not_in, orderby, order.
func applyRestPostSearchQuery(c *wp.Context, r *http.Request, postTypes []string, term string, include, exclude []int64, orderby, order string) ([]string, string, []int64, []int64, string, string) {
	if c == nil || c.Hooks == nil {
		return postTypes, term, include, exclude, orderby, order
	}
	args := map[string]any{
		"s":            term,
		"post_type":    append([]string(nil), postTypes...),
		"post__in":     append([]int64(nil), include...),
		"post__not_in": append([]int64(nil), exclude...),
		"orderby":      orderby,
		"order":        order,
	}
	filtered := c.Hooks.ApplyFilters("rest_post_search_query", args, r)
	m, ok := filtered.(map[string]any)
	if !ok || m == nil {
		return postTypes, term, include, exclude, orderby, order
	}
	if s, ok := m["s"].(string); ok {
		term = s
	}
	if pts, ok := m["post_type"].([]string); ok && len(pts) > 0 {
		postTypes = pts
	}
	if in, ok := m["post__in"].([]int64); ok {
		include = in
	}
	if ex, ok := m["post__not_in"].([]int64); ok {
		exclude = ex
	}
	if ob, ok := m["orderby"].(string); ok && ob != "" {
		orderby = ob
	}
	if o, ok := m["order"].(string); ok && o != "" {
		order = o
	}
	return postTypes, term, include, exclude, orderby, order
}

func countSearch(c *wp.Context, term string, postTypes []string, include, exclude []int64) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	if len(postTypes) == 0 {
		postTypes = append([]string(nil), publicRESTPostSearchTypes...)
	}
	like := "%" + db.EscapeLike(term) + "%"
	args := make([]any, 0, len(postTypes)+len(include)+len(exclude)+3)
	args = append(args, "publish")
	inParts := make([]string, len(postTypes))
	for i, pt := range postTypes {
		inParts[i] = "%s"
		args = append(args, pt)
	}
	whereExtra, args := searchIncludeExcludeSQL("ID", include, exclude, args)
	args = append(args, like, like, like)
	q, prepArgs, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.Posts()+
			" WHERE post_status = %s AND post_type IN ("+strings.Join(inParts, ", ")+
			")"+whereExtra+" AND (post_title LIKE %s OR post_content LIKE %s OR post_excerpt LIKE %s)",
		args...,
	)
	if err != nil {
		return 0, err
	}
	row, err := c.DB.GetRow(q, prepArgs...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return int(asInt64(row["total"])), nil
}

// searchIncludeExcludeSQL appends post__in / post__not_in style clauses (P4.100).
func searchIncludeExcludeSQL(column string, include, exclude []int64, args []any) (string, []any) {
	var extra strings.Builder
	if len(include) > 0 {
		parts := make([]string, len(include))
		for i, id := range include {
			parts[i] = "%d"
			args = append(args, id)
		}
		extra.WriteString(" AND " + column + " IN (" + strings.Join(parts, ", ") + ")")
	}
	if len(exclude) > 0 {
		parts := make([]string, len(exclude))
		for i, id := range exclude {
			parts[i] = "%d"
			args = append(args, id)
		}
		extra.WriteString(" AND " + column + " NOT IN (" + strings.Join(parts, ", ") + ")")
	}
	return extra.String(), args
}

func searchFromRow(c *wp.Context, row map[string]any, home string) SearchResult {
	id := asInt64(row["ID"])
	postType := asString(row["post_type"])
	slug := asString(row["post_name"])
	dateStr := asString(row["post_date"])
	p := wp.Post{
		ID: id, Type: postType, Name: slug, Date: dateStr,
		Parent: asInt64(row["post_parent"]),
	}
	return SearchResult{
		ID:      id,
		Title:   asString(row["post_title"]),
		URL:     itemPermalink(c, home, p),
		Type:    "post",
		Subtype: postType,
		Links:   buildSearchLinks(home, id, postType),
	}
}

func buildSearchLinks(home string, id int64, postType string) map[string]any {
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	base := "posts"
	if postType == "page" {
		base = "pages"
	}
	return map[string]any{
		"self": []map[string]any{{
			"embeddable":  true,
			"href":        fmt.Sprintf("%s/%s/%d", restRoot, base, id),
			"targetHints": map[string]any{"allow": []string{"GET"}},
		}},
		"about":      []map[string]any{{"href": restRoot + "/types/" + postType}},
		"collection": []map[string]any{{"href": restRoot + "/search"}},
	}
}

// --- term search (P4.49) ---

var publicRESTTaxonomies = []string{"category", "post_tag"}

func resolveTermTaxonomies(subtype []string) []string {
	if len(subtype) == 0 {
		return append([]string(nil), publicRESTTaxonomies...)
	}
	allowed := map[string]bool{}
	for _, t := range publicRESTTaxonomies {
		allowed[t] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, s := range subtype {
		if !allowed[s] || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 {
		return append([]string(nil), publicRESTTaxonomies...)
	}
	return out
}

func listTermSearchPaged(c *wp.Context, term string, subtype []string, include, exclude []int64, orderby, order string, limit, page int) ([]SearchResult, error) {
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
	taxonomies := resolveTermTaxonomies(subtype)
	like := "%" + db.EscapeLike(term) + "%"

	// Build IN list via repeated placeholders.
	args := make([]any, 0, len(taxonomies)+len(include)+len(exclude)+5)
	inParts := make([]string, len(taxonomies))
	for i, tax := range taxonomies {
		inParts[i] = "%s"
		args = append(args, tax)
	}
	whereExtra, args := searchIncludeExcludeSQL("t.term_id", include, exclude, args)
	args = append(args, like)
	orderSQL, orderArgs := searchTermOrderSQL(orderby, order, include)
	args = append(args, orderArgs...)
	args = append(args, limit, offset)
	q, prepArgs, err := c.DB.Prepare(
		"SELECT t.term_id, t.name, t.slug, tt.taxonomy, tt.parent FROM "+
			c.DB.Terms()+" t INNER JOIN "+c.DB.TermTaxonomy()+
			" tt ON t.term_id = tt.term_id WHERE tt.taxonomy IN ("+strings.Join(inParts, ", ")+
			")"+whereExtra+" AND t.name LIKE %s ORDER BY "+orderSQL+" LIMIT %d OFFSET %d",
		args...,
	)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(q, prepArgs...)
	if err != nil {
		return nil, err
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	out := make([]SearchResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, termSearchFromRow(c, row, home))
	}
	return out, nil
}

func countTermSearch(c *wp.Context, term string, subtype []string, include, exclude []int64) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	taxonomies := resolveTermTaxonomies(subtype)
	like := "%" + db.EscapeLike(term) + "%"
	args := make([]any, 0, len(taxonomies)+len(include)+len(exclude)+1)
	inParts := make([]string, len(taxonomies))
	for i, tax := range taxonomies {
		inParts[i] = "%s"
		args = append(args, tax)
	}
	whereExtra, args := searchIncludeExcludeSQL("t.term_id", include, exclude, args)
	args = append(args, like)
	q, prepArgs, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.Terms()+" t INNER JOIN "+c.DB.TermTaxonomy()+
			" tt ON t.term_id = tt.term_id WHERE tt.taxonomy IN ("+strings.Join(inParts, ", ")+
			")"+whereExtra+" AND t.name LIKE %s",
		args...,
	)
	if err != nil {
		return 0, err
	}
	row, err := c.DB.GetRow(q, prepArgs...)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}
	return int(asInt64(row["total"])), nil
}

// searchTermOrderSQL builds ORDER BY for term search (P4.122).
func searchTermOrderSQL(orderby, order string, include []int64) (string, []any) {
	dir := "ASC"
	if order == "desc" {
		dir = "DESC"
	}
	switch orderby {
	case "id":
		return "t.term_id " + dir, nil
	case "slug":
		return "t.slug " + dir + ", t.name ASC", nil
	case "include":
		if len(include) == 0 {
			return "t.name ASC", nil
		}
		parts := make([]string, len(include))
		args := make([]any, len(include))
		for i, id := range include {
			parts[i] = "%d"
			args[i] = id
		}
		return "FIELD(t.term_id, " + strings.Join(parts, ", ") + ")", args
	default: // name
		return "t.name " + dir, nil
	}
}

func termSearchFromRow(c *wp.Context, row map[string]any, home string) SearchResult {
	id := asInt64(row["term_id"])
	taxonomy := asString(row["taxonomy"])
	slug := asString(row["slug"])
	parent := asInt64(row["parent"])
	return SearchResult{
		ID:    id,
		Title: asString(row["name"]),
		URL:   termPermalink(c, home, taxonomy, id, slug, parent),
		Type:  taxonomy, // WP_REST_Term_Search_Handler sets type to taxonomy name
		Links: buildTermSearchLinks(home, taxonomy, id),
	}
}

func buildTermSearchLinks(home, taxonomy string, id int64) map[string]any {
	restRoot := strings.TrimRight(home, "/") + "/wp-json/wp/v2"
	base := "tags"
	if taxonomy == "category" {
		base = "categories"
	}
	return map[string]any{
		"self": []map[string]any{{
			"embeddable":  true,
			"href":        fmt.Sprintf("%s/%s/%d", restRoot, base, id),
			"targetHints": map[string]any{"allow": []string{"GET"}},
		}},
		"about":      []map[string]any{{"href": restRoot + "/taxonomies/" + taxonomy}},
		"collection": []map[string]any{{"href": restRoot + "/search"}},
	}
}

// --- post-format search (P4.49) ---

var postFormatStrings = []struct {
	slug  string
	title string
}{
	{"standard", "Standard"},
	{"aside", "Aside"},
	{"chat", "Chat"},
	{"gallery", "Gallery"},
	{"link", "Link"},
	{"image", "Image"},
	{"quote", "Quote"},
	{"status", "Status"},
	{"video", "Video"},
	{"audio", "Audio"},
}

func listPostFormatSearch(c *wp.Context, term string, limit, page int) (int, []SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	home := strings.TrimRight(optionString(c, "home", optionString(c, "siteurl", "")), "/")
	existing := postFormatTermSlugs(c)
	needle := strings.ToLower(strings.TrimSpace(term))

	var matched []SearchResult
	for _, pf := range postFormatStrings {
		if pf.slug == "standard" {
			continue // standard has no archive term / link
		}
		if !existing[pf.slug] {
			continue
		}
		if needle != "" {
			if !strings.Contains(strings.ToLower(pf.slug), needle) &&
				!strings.Contains(strings.ToLower(pf.title), needle) {
				continue
			}
		}
		matched = append(matched, SearchResult{
			ID:    pf.slug,
			Title: pf.title,
			URL:   home + "/type/" + pf.slug + "/",
			Type:  "post-format",
			Links: map[string]any{
				"collection": []map[string]any{{"href": strings.TrimRight(home, "/") + "/wp-json/wp/v2/search"}},
			},
		})
	}
	total := len(matched)
	offset := (page - 1) * limit
	if offset >= total {
		return total, []SearchResult{}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return total, matched[offset:end], nil
}

func postFormatTermSlugs(c *wp.Context) map[string]bool {
	out := map[string]bool{}
	if c == nil || c.DB == nil {
		return out
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.slug FROM "+c.DB.Terms()+" t INNER JOIN "+c.DB.TermTaxonomy()+
			" tt ON t.term_id = tt.term_id WHERE tt.taxonomy = %s",
		"post_format",
	)
	if err != nil {
		return out
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return out
	}
	for _, row := range rows {
		slug := asString(row["slug"])
		slug = strings.TrimPrefix(slug, "post-format-")
		if slug != "" {
			out[slug] = true
		}
	}
	return out
}
