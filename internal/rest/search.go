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

	var (
		total   int
		results []SearchResult
		err     error
	)
	switch searchType {
	case "term":
		total, err = countTermSearch(c, term, subtype)
		if err == nil {
			results, err = listTermSearchPaged(c, term, subtype, perPage, page)
		}
	case "post-format":
		total, results, err = listPostFormatSearch(c, term, perPage, page)
	default:
		postTypes := resolvePostSearchSubtypes(subtype)
		total, err = countSearch(c, term, postTypes)
		if err == nil {
			results, err = listSearchPaged(c, term, postTypes, perPage, page)
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

func listSearchPaged(c *wp.Context, term string, postTypes []string, limit, page int) ([]SearchResult, error) {
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
	args := make([]any, 0, len(postTypes)+4)
	args = append(args, "publish")
	inParts := make([]string, len(postTypes))
	for i, pt := range postTypes {
		inParts[i] = "%s"
		args = append(args, pt)
	}
	args = append(args, like, like, limit, offset)
	q, prepArgs, err := c.DB.Prepare(
		"SELECT ID, post_date, post_name, post_title, post_type, post_parent FROM "+c.DB.Posts()+
			" WHERE post_status = %s AND post_type IN ("+strings.Join(inParts, ", ")+
			") AND (post_title LIKE %s OR post_content LIKE %s) "+
			"ORDER BY post_date DESC LIMIT %d OFFSET %d",
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

func countSearch(c *wp.Context, term string, postTypes []string) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	if len(postTypes) == 0 {
		postTypes = append([]string(nil), publicRESTPostSearchTypes...)
	}
	like := "%" + db.EscapeLike(term) + "%"
	args := make([]any, 0, len(postTypes)+3)
	args = append(args, "publish")
	inParts := make([]string, len(postTypes))
	for i, pt := range postTypes {
		inParts[i] = "%s"
		args = append(args, pt)
	}
	args = append(args, like, like)
	q, prepArgs, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.Posts()+
			" WHERE post_status = %s AND post_type IN ("+strings.Join(inParts, ", ")+
			") AND (post_title LIKE %s OR post_content LIKE %s)",
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

func listTermSearchPaged(c *wp.Context, term string, subtype []string, limit, page int) ([]SearchResult, error) {
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
	args := make([]any, 0, len(taxonomies)+3)
	inParts := make([]string, len(taxonomies))
	for i, tax := range taxonomies {
		inParts[i] = "%s"
		args = append(args, tax)
	}
	args = append(args, like, limit, offset)
	q, prepArgs, err := c.DB.Prepare(
		"SELECT t.term_id, t.name, t.slug, tt.taxonomy, tt.parent FROM "+
			c.DB.Terms()+" t INNER JOIN "+c.DB.TermTaxonomy()+
			" tt ON t.term_id = tt.term_id WHERE tt.taxonomy IN ("+strings.Join(inParts, ", ")+
			") AND t.name LIKE %s ORDER BY t.name ASC LIMIT %d OFFSET %d",
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

func countTermSearch(c *wp.Context, term string, subtype []string) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("nil DB")
	}
	taxonomies := resolveTermTaxonomies(subtype)
	like := "%" + db.EscapeLike(term) + "%"
	args := make([]any, 0, len(taxonomies)+1)
	inParts := make([]string, len(taxonomies))
	for i, tax := range taxonomies {
		inParts[i] = "%s"
		args = append(args, tax)
	}
	args = append(args, like)
	q, prepArgs, err := c.DB.Prepare(
		"SELECT COUNT(*) AS total FROM "+c.DB.Terms()+" t INNER JOIN "+c.DB.TermTaxonomy()+
			" tt ON t.term_id = tt.term_id WHERE tt.taxonomy IN ("+strings.Join(inParts, ", ")+
			") AND t.name LIKE %s",
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
