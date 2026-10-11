// Package wp — WP_Query-like post helpers (P3.6).
package wp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/db"
)

// QueryArgs is a minimal subset of WP_Query / get_posts arguments.
type QueryArgs struct {
	PostType   string // default "post"
	PostStatus string // default "publish"; ignored when PostStatuses is set
	// PostStatuses is WP_Query post_status as a list (REST ?status=, P4.88).
	// A single value "any" means all statuses except trash and auto-draft.
	PostStatuses []string
	PostsPerPage int    // default 10; -1 means no LIMIT
	Offset       int    // SQL OFFSET (ignored when PostsPerPage < 0)
	P            int64  // post ID (query var "p")
	Name         string // post slug (query var "name")
	// NameIn is post_name__in (REST ?slug=, P4.89).
	NameIn   []string
	PageName string // page slug (query var "pagename"); implies post_type=page when PostType unset
	// Search is the `s` query var (title/content LIKE, P3.21).
	Search string
	// SearchColumns limits Search LIKE columns (REST search_columns, P4.135).
	// Allowed: post_title, post_content, post_excerpt. Empty → all three.
	SearchColumns []string
	// Year / MonthNum / Day filter post_date for date archives (P3.22).
	Year     int
	MonthNum int
	Day      int
	// Author is the post_author user ID for author archives (P3.23).
	Author int64
	// AuthorIn is author__in (REST ?author= list, P4.85 / P4.132).
	AuthorIn []int64
	// AuthorNotIn is author__not_in (REST ?author_exclude=, P4.132).
	AuthorNotIn []int64
	// Tag is the post_tag slug for tag archives (P3.24).
	Tag string
	// Category is the category slug for category archives (P3.25).
	Category string
	// CategoryIn is category term IDs (WP_Query category__in / REST ?categories=, P4.86).
	CategoryIn []int64
	// CategoryNotIn is category term IDs to exclude (REST ?categories_exclude=, P4.136).
	CategoryNotIn []int64
	// TagIn is post_tag term IDs (WP_Query tag__in / REST ?tags=, P4.87).
	TagIn []int64
	// TagNotIn is post_tag term IDs to exclude (REST ?tags_exclude=, P4.136).
	TagNotIn []int64
	// TaxRelation is tax_query relation AND|OR across category/tag clauses (REST ?tax_relation=, P4.136).
	// Empty defaults to AND.
	TaxRelation string
	// PostFormatIn is post_format term slugs with post-format- prefix (REST ?format=, P4.96).
	PostFormatIn []string
	// PostFormatStandard includes posts with no post_format term (format=standard, P4.96).
	PostFormatStandard bool
	// StickyFirst prepends sticky posts (home/blog inherit queries, P5.19).
	StickyFirst bool
	// IgnoreSticky excludes sticky posts from results (non-inherit query loops).
	IgnoreSticky bool
	// PostIn limits results to these IDs (post__in; REST sticky=true, P4.82).
	PostIn []int64
	// PostNotIn excludes these IDs (post__not_in; REST sticky=false, P4.82).
	PostNotIn []int64
	// OrderBy is a WP_Query orderby alias: date (default), title, id, slug,
	// menu_order, author, modified, parent, include, include_slugs, relevance
	// (P4.83 / P4.94 / P4.131).
	OrderBy string
	// Order is ASC or DESC (default DESC) (P4.83).
	Order string
	// DateAfter / DateBefore filter post_date (REST ?after=/?before=, inclusive, P4.91).
	// Values are MySQL datetimes "2006-01-02 15:04:05".
	DateAfter  string
	DateBefore string
	// ModifiedAfter / ModifiedBefore filter post_modified (REST ?modified_after=/?modified_before=, P4.95).
	ModifiedAfter  string
	ModifiedBefore string
	// ParentIn is post_parent__in (REST ?parent= for hierarchical types, P4.93).
	// Includes 0 for top-level pages.
	ParentIn []int64
	// ParentNotIn is post_parent__not_in (REST ?parent_exclude=, P4.93).
	ParentNotIn []int64
}

// Post is a row from wp_posts used by front and REST helpers.
type Post struct {
	ID            int64
	Author        int64
	Title         string
	Content       string
	Excerpt       string
	Name          string
	Status        string
	Type          string
	Date          string
	DateGMT       string
	Modified      string
	ModifiedGMT   string
	GUID          string
	CommentStatus string
	PingStatus    string
	Password      string
	Parent        int64
	MenuOrder     int
}

// CountPosts returns the number of posts matching args (ignores Limit/Offset).
// Used for query-pagination max_num_pages (P5.44).
func CountPosts(c *Context, args QueryArgs) (int, error) {
	if c == nil || c.DB == nil {
		return 0, fmt.Errorf("wp: CountPosts: nil DB")
	}
	postType := args.PostType
	if postType == "" {
		if args.PageName != "" {
			postType = "page"
		} else {
			postType = "post"
		}
	}
	var where []string
	var vals []any
	where = append(where, "post_type = %s")
	vals = append(vals, postType)
	if clause, cvals := postStatusSQL(args.PostStatus, args.PostStatuses); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if clause, cvals := postNameFilterSQL(args.Name, args.PageName, args.NameIn); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if clause, cvals := searchFilterSQL(args.Search, args.SearchColumns); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if args.Year > 0 {
		where = append(where, "YEAR(post_date) = %d")
		vals = append(vals, args.Year)
	}
	if args.MonthNum > 0 {
		where = append(where, "MONTH(post_date) = %d")
		vals = append(vals, args.MonthNum)
	}
	if args.Day > 0 {
		where = append(where, "DAY(post_date) = %d")
		vals = append(vals, args.Day)
	}
	where, vals = appendDateQueryFilters(where, vals, args)
	if clause, cvals := authorFilterSQL(args); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if clause, cvals := postInNotInSQL("ID", args.PostIn, args.PostNotIn); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if clause, cvals := postParentInNotInSQL(args.ParentIn, args.ParentNotIn); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	from := c.DB.Posts()
	if tag := strings.TrimSpace(args.Tag); tag != "" {
		from = c.DB.Posts() + " p INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s", "t.slug = %s")
		vals = append(vals, "post_tag", tag)
		where, vals = appendTaxQueryFilters(c, where, vals, args, "p.ID")
		where, vals = appendPostFormatFilter(c, where, vals, args, "p.ID")
		whereSQL := strings.Join(where, " AND ")
		whereSQL = strings.ReplaceAll(whereSQL, "post_type", "p.post_type")
		whereSQL = strings.ReplaceAll(whereSQL, "post_status", "p.post_status")
		whereSQL = strings.ReplaceAll(whereSQL, "post_author", "p.post_author")
		whereSQL = strings.ReplaceAll(whereSQL, "post_title", "p.post_title")
		whereSQL = strings.ReplaceAll(whereSQL, "post_excerpt", "p.post_excerpt")
		whereSQL = strings.ReplaceAll(whereSQL, "post_content", "p.post_content")
		whereSQL = strings.ReplaceAll(whereSQL, "post_name", "p.post_name")
		whereSQL = strings.ReplaceAll(whereSQL, "post_date", "p.post_date")
		whereSQL = strings.ReplaceAll(whereSQL, "post_modified", "p.post_modified")
		whereSQL = strings.ReplaceAll(whereSQL, "ID IN", "p.ID IN")
		whereSQL = strings.ReplaceAll(whereSQL, "ID NOT IN", "p.ID NOT IN")
		whereSQL = strings.ReplaceAll(whereSQL, "post_parent", "p.post_parent")
		q := "SELECT COUNT(DISTINCT p.ID) AS total FROM " + from + " WHERE " + whereSQL
		sqlStr, sqlArgs, err := c.DB.Prepare(q, vals...)
		if err != nil {
			return 0, err
		}
		v, err := c.DB.GetVar(sqlStr, sqlArgs...)
		if err != nil {
			return 0, err
		}
		return intFromAny(v), nil
	}
	if cat := strings.TrimSpace(args.Category); cat != "" {
		cat = lastPathSegment(cat)
		from = c.DB.Posts() + " p INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s", "t.slug = %s")
		vals = append(vals, "category", cat)
		where, vals = appendTaxQueryFilters(c, where, vals, args, "p.ID")
		where, vals = appendPostFormatFilter(c, where, vals, args, "p.ID")
		whereSQL := strings.Join(where, " AND ")
		whereSQL = strings.ReplaceAll(whereSQL, "post_type", "p.post_type")
		whereSQL = strings.ReplaceAll(whereSQL, "post_status", "p.post_status")
		whereSQL = strings.ReplaceAll(whereSQL, "post_author", "p.post_author")
		whereSQL = strings.ReplaceAll(whereSQL, "post_title", "p.post_title")
		whereSQL = strings.ReplaceAll(whereSQL, "post_excerpt", "p.post_excerpt")
		whereSQL = strings.ReplaceAll(whereSQL, "post_content", "p.post_content")
		whereSQL = strings.ReplaceAll(whereSQL, "post_name", "p.post_name")
		whereSQL = strings.ReplaceAll(whereSQL, "post_date", "p.post_date")
		whereSQL = strings.ReplaceAll(whereSQL, "post_modified", "p.post_modified")
		whereSQL = strings.ReplaceAll(whereSQL, "ID IN", "p.ID IN")
		whereSQL = strings.ReplaceAll(whereSQL, "ID NOT IN", "p.ID NOT IN")
		whereSQL = strings.ReplaceAll(whereSQL, "post_parent", "p.post_parent")
		q := "SELECT COUNT(DISTINCT p.ID) AS total FROM " + from + " WHERE " + whereSQL
		sqlStr, sqlArgs, err := c.DB.Prepare(q, vals...)
		if err != nil {
			return 0, err
		}
		v, err := c.DB.GetVar(sqlStr, sqlArgs...)
		if err != nil {
			return 0, err
		}
		return intFromAny(v), nil
	}
	where, vals = appendTaxQueryFilters(c, where, vals, args, "ID")
	where, vals = appendPostFormatFilter(c, where, vals, args, "ID")
	q := "SELECT COUNT(*) AS total FROM " + from + " WHERE " + strings.Join(where, " AND ")
	sqlStr, sqlArgs, err := c.DB.Prepare(q, vals...)
	if err != nil {
		return 0, err
	}
	v, err := c.DB.GetVar(sqlStr, sqlArgs...)
	if err != nil {
		return 0, err
	}
	return intFromAny(v), nil
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case int64:
		return int(t)
	case int:
		return t
	case uint64:
		return int(t)
	case []byte:
		n, _ := strconv.Atoi(string(t))
		return n
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		n, _ := strconv.Atoi(fmt.Sprint(t))
		return n
	}
}

// GetPosts runs a WP_Query-like SELECT against wp_posts.
func GetPosts(c *Context, args QueryArgs) ([]Post, error) {
	if c == nil || c.DB == nil {
		return nil, fmt.Errorf("wp: GetPosts: nil DB")
	}

	postType := args.PostType
	if postType == "" {
		if args.PageName != "" {
			postType = "page"
		} else {
			postType = "post"
		}
	}
	limit := args.PostsPerPage
	if limit == 0 {
		limit = 10
	}

	var where []string
	var vals []any

	where = append(where, "post_type = %s")
	vals = append(vals, postType)
	if clause, cvals := postStatusSQL(args.PostStatus, args.PostStatuses); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}

	if args.P > 0 {
		where = append(where, "ID = %d")
		vals = append(vals, args.P)
	}
	if clause, cvals := postNameFilterSQL(args.Name, args.PageName, args.NameIn); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if clause, cvals := searchFilterSQL(args.Search, args.SearchColumns); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if args.Year > 0 {
		where = append(where, "YEAR(post_date) = %d")
		vals = append(vals, args.Year)
	}
	if args.MonthNum > 0 {
		where = append(where, "MONTH(post_date) = %d")
		vals = append(vals, args.MonthNum)
	}
	if args.Day > 0 {
		where = append(where, "DAY(post_date) = %d")
		vals = append(vals, args.Day)
	}
	where, vals = appendDateQueryFilters(where, vals, args)
	if clause, cvals := authorFilterSQL(args); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if clause, cvals := postInNotInSQL("ID", args.PostIn, args.PostNotIn); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	if clause, cvals := postParentInNotInSQL(args.ParentIn, args.ParentNotIn); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	from := c.DB.Posts() + " p"
	if tag := strings.TrimSpace(args.Tag); tag != "" {
		from += " INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s", "t.slug = %s")
		vals = append(vals, "post_tag", tag)
	} else if cat := strings.TrimSpace(args.Category); cat != "" {
		cat = lastPathSegment(cat)
		from += " INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s", "t.slug = %s")
		vals = append(vals, "category", cat)
	}

	// Qualifying column names when joining taxonomies.
	aliased := strings.Contains(from, " tr ")
	idCol := "ID"
	if aliased {
		idCol = "p.ID"
	} else {
		from = c.DB.Posts()
	}
	// P4.136: categories/tags include+exclude via EXISTS (combine with tax_relation).
	where, vals = appendTaxQueryFilters(c, where, vals, args, idCol)
	// P4.96: format= via EXISTS / NOT EXISTS (works with or without tax joins).
	where, vals = appendPostFormatFilter(c, where, vals, args, idCol)
	whereSQL := strings.Join(where, " AND ")
	if aliased {
		whereSQL = strings.ReplaceAll(whereSQL, "post_type", "p.post_type")
		whereSQL = strings.ReplaceAll(whereSQL, "post_status", "p.post_status")
		whereSQL = strings.ReplaceAll(whereSQL, "post_author", "p.post_author")
		whereSQL = strings.ReplaceAll(whereSQL, "post_title", "p.post_title")
		whereSQL = strings.ReplaceAll(whereSQL, "post_excerpt", "p.post_excerpt")
		whereSQL = strings.ReplaceAll(whereSQL, "post_content", "p.post_content")
		whereSQL = strings.ReplaceAll(whereSQL, "post_name", "p.post_name")
		whereSQL = strings.ReplaceAll(whereSQL, "post_date", "p.post_date")
		whereSQL = strings.ReplaceAll(whereSQL, "post_modified", "p.post_modified")
		whereSQL = strings.ReplaceAll(whereSQL, "ID =", "p.ID =")
		whereSQL = strings.ReplaceAll(whereSQL, "ID IN", "p.ID IN")
		whereSQL = strings.ReplaceAll(whereSQL, "ID NOT IN", "p.ID NOT IN")
		whereSQL = strings.ReplaceAll(whereSQL, "post_parent", "p.post_parent")
	}

	orderSQL, orderVals := orderByClause(args, aliased)
	vals = append(vals, orderVals...)
	q := "SELECT " + selectPostColumns(aliased) + " FROM " +
		from + " WHERE " + whereSQL +
		" ORDER BY " + orderSQL
	if limit >= 0 {
		q += " LIMIT %d"
		vals = append(vals, limit)
		if args.Offset > 0 {
			q += " OFFSET %d"
			vals = append(vals, args.Offset)
		}
	}

	sqlStr, sqlArgs, err := c.DB.Prepare(q, vals...)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.GetResults(sqlStr, sqlArgs...)
	if err != nil {
		return nil, err
	}
	out := make([]Post, 0, len(rows))
	for _, row := range rows {
		out = append(out, postFromRow(row))
	}
	if postType == "post" && (args.StickyFirst || args.IgnoreSticky) && args.P == 0 && args.Name == "" && args.PageName == "" && len(args.NameIn) == 0 && args.Tag == "" && args.Category == "" && len(args.CategoryIn) == 0 && len(args.CategoryNotIn) == 0 && len(args.TagIn) == 0 && len(args.TagNotIn) == 0 && len(args.PostFormatIn) == 0 && !args.PostFormatStandard {
		out = applyStickyOrdering(c, out, args)
	}
	// P9.28: posts_results / the_posts via apply_filters_ref_array (query + posts slice).
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFiltersRefArray("posts_results", []any{out, args}); v != nil {
			if posts, ok := v.([]Post); ok {
				out = posts
			}
		}
		if v := c.Hooks.ApplyFiltersRefArray("the_posts", []any{out, args}); v != nil {
			if posts, ok := v.([]Post); ok {
				out = posts
			}
		}
	}
	return out, nil
}

func applyStickyOrdering(c *Context, posts []Post, args QueryArgs) []Post {
	sticky := stickyPostIDs(c)
	if len(sticky) == 0 {
		return posts
	}
	stickySet := map[int64]bool{}
	for _, id := range sticky {
		stickySet[id] = true
	}
	if args.IgnoreSticky {
		out := make([]Post, 0, len(posts))
		for _, p := range posts {
			if !stickySet[p.ID] {
				out = append(out, p)
			}
		}
		return out
	}
	if !args.StickyFirst {
		return posts
	}
	// Prepend sticky posts that are published, then non-sticky in date order.
	seen := map[int64]bool{}
	var head []Post
	for _, id := range sticky {
		if seen[id] {
			continue
		}
		p, err := GetPostByID(c, id)
		if err != nil || p == nil || p.Status != "publish" || p.Type != "post" {
			continue
		}
		head = append(head, *p)
		seen[id] = true
	}
	var rest []Post
	for _, p := range posts {
		if seen[p.ID] {
			continue
		}
		rest = append(rest, p)
	}
	// Sticky posts on the main blog query are shown in addition to posts_per_page.
	limit := args.PostsPerPage
	if limit <= 0 {
		return append(head, rest...)
	}
	max := limit + len(head)
	out := append(head, rest...)
	if len(out) > max {
		out = out[:max]
	}
	return out
}

func stickyPostIDs(c *Context) []int64 {
	if c == nil {
		return nil
	}
	raw := ""
	if c.Options != nil {
		if v := c.Options.GetOption("sticky_posts", nil); v != nil {
			raw = fmt.Sprint(v)
		}
	}
	if raw == "" || raw == "<nil>" {
		return nil
	}
	// PHP serialized: a:1:{i:0;i:13;}
	if strings.Contains(raw, "a:") {
		return parsePHPIntArray(raw)
	}
	return nil
}

func parsePHPIntArray(ser string) []int64 {
	// Match i:<digits>; that are values in a:N:{i:k;i:val;...}
	re := regexp.MustCompile(`i:(\d+);i:(\d+);`)
	matches := re.FindAllStringSubmatch(ser, -1)
	out := make([]int64, 0, len(matches))
	seen := map[int64]bool{}
	for _, m := range matches {
		n, _ := strconv.ParseInt(m[2], 10, 64)
		if n > 0 && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func selectPostColumns(aliased bool) string {
	cols := "ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, " +
		"post_name, post_status, post_type, post_title, post_content, post_excerpt, " +
		"guid, comment_status, ping_status, post_password, post_parent, menu_order"
	if !aliased {
		return cols
	}
	parts := strings.Split(cols, ", ")
	for i, p := range parts {
		parts[i] = "p." + p
	}
	return strings.Join(parts, ", ")
}

func orderByPostDate(aliased bool) string {
	if aliased {
		return "p.post_date"
	}
	return "post_date"
}

// appendDateQueryFilters applies REST after/before / modified_* filters (P4.91).
// Comparisons are inclusive to match WP date_query inclusive=true.
func appendDateQueryFilters(where []string, vals []any, args QueryArgs) ([]string, []any) {
	if s := strings.TrimSpace(args.DateAfter); s != "" {
		where = append(where, "post_date >= %s")
		vals = append(vals, s)
	}
	if s := strings.TrimSpace(args.DateBefore); s != "" {
		where = append(where, "post_date <= %s")
		vals = append(vals, s)
	}
	if s := strings.TrimSpace(args.ModifiedAfter); s != "" {
		where = append(where, "post_modified >= %s")
		vals = append(vals, s)
	}
	if s := strings.TrimSpace(args.ModifiedBefore); s != "" {
		where = append(where, "post_modified <= %s")
		vals = append(vals, s)
	}
	return where, vals
}

// orderByClause maps QueryArgs.OrderBy/Order onto SQL (P4.83 / P4.94 / P4.131).
// Supported orderby: date (default), title, id, slug, menu_order, author,
// modified, parent, include (FIELD post__in), include_slugs (FIELD post_name__in),
// relevance (title then date when searching).
// Order: ASC|DESC (default DESC). Extra Prepare args are returned for FIELD lists.
func orderByClause(args QueryArgs, aliased bool) (string, []any) {
	orderby := strings.ToLower(strings.TrimSpace(args.OrderBy))
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(args.Order), "ASC") {
		dir = "ASC"
	}
	idCol := "ID"
	titleCol := "post_title"
	nameCol := "post_name"
	authorCol := "post_author"
	modifiedCol := "post_modified"
	parentCol := "post_parent"
	menuCol := "menu_order"
	dateCol := orderByPostDate(aliased)
	if aliased {
		idCol = "p.ID"
		titleCol = "p.post_title"
		nameCol = "p.post_name"
		authorCol = "p.post_author"
		modifiedCol = "p.post_modified"
		parentCol = "p.post_parent"
		menuCol = "p.menu_order"
	}
	switch orderby {
	case "title":
		return titleCol + " " + dir, nil
	case "id":
		return idCol + " " + dir, nil
	case "slug", "post_name", "name":
		return nameCol + " " + dir, nil
	case "menu_order":
		// P4.94: REST pages orderby=menu_order (page-attributes).
		return menuCol + " " + dir, nil
	case "author":
		return authorCol + " " + dir + ", " + idCol + " " + dir, nil
	case "modified", "post_modified":
		return modifiedCol + " " + dir + ", " + idCol + " " + dir, nil
	case "parent":
		return parentCol + " " + dir + ", " + idCol + " " + dir, nil
	case "include":
		if len(args.PostIn) == 0 {
			return dateCol + " " + dir, nil
		}
		parts := make([]string, len(args.PostIn))
		vals := make([]any, len(args.PostIn))
		for i, id := range args.PostIn {
			parts[i] = "%d"
			vals[i] = id
		}
		return "FIELD(" + idCol + ", " + strings.Join(parts, ", ") + ")", vals
	case "include_slugs":
		if len(args.NameIn) == 0 {
			return dateCol + " " + dir, nil
		}
		parts := make([]string, len(args.NameIn))
		vals := make([]any, len(args.NameIn))
		for i, slug := range args.NameIn {
			parts[i] = "%s"
			vals[i] = slug
		}
		return "FIELD(" + nameCol + ", " + strings.Join(parts, ", ") + ")", vals
	case "relevance":
		// Without FULLTEXT, approximate WP search relevance: title then date.
		return titleCol + " ASC, " + dateCol + " DESC, " + idCol + " ASC", nil
	case "date", "post_date", "":
		return dateCol + " " + dir, nil
	default:
		// Unknown aliases fall back to date (safe default).
		return dateCol + " " + dir, nil
	}
}

// postStatusSQL builds a post_status WHERE fragment (P4.88).
// PostStatuses takes precedence; empty both defaults to publish.
// A lone "any" excludes trash and auto-draft (WP_Query post_status=any).
func postStatusSQL(status string, statuses []string) (string, []any) {
	list := statuses
	if len(list) == 0 {
		s := strings.TrimSpace(status)
		if s == "" {
			s = "publish"
		}
		list = []string{s}
	}
	if len(list) == 1 && strings.EqualFold(list[0], "any") {
		return "post_status NOT IN (%s, %s)", []any{"trash", "auto-draft"}
	}
	if len(list) == 1 {
		return "post_status = %s", []any{list[0]}
	}
	ph := make([]string, len(list))
	vals := make([]any, len(list))
	for i, s := range list {
		ph[i] = "%s"
		vals[i] = s
	}
	return "post_status IN (" + strings.Join(ph, ", ") + ")", vals
}

// authorFilterSQL builds post_author / author__in / author__not_in (P3.23 / P4.132).
// AuthorIn takes precedence over single Author when non-empty.
func authorFilterSQL(args QueryArgs) (string, []any) {
	var parts []string
	var vals []any
	if len(args.AuthorIn) > 0 {
		ph := make([]string, len(args.AuthorIn))
		for i, id := range args.AuthorIn {
			ph[i] = "%d"
			vals = append(vals, id)
		}
		parts = append(parts, "post_author IN ("+strings.Join(ph, ",")+")")
	} else if args.Author > 0 {
		parts = append(parts, "post_author = %d")
		vals = append(vals, args.Author)
	}
	if len(args.AuthorNotIn) > 0 {
		ph := make([]string, len(args.AuthorNotIn))
		for i, id := range args.AuthorNotIn {
			ph[i] = "%d"
			vals = append(vals, id)
		}
		parts = append(parts, "post_author NOT IN ("+strings.Join(ph, ",")+")")
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " AND "), vals
}

// postInNotInSQL builds post__in / post__not_in fragments (P4.82).
// Empty PostIn yields "ID IN (0)" so WP_Query returns no rows.
// postNameFilterSQL builds post_name = / post_name IN (…) for name/pagename/NameIn (P4.89).
func postNameFilterSQL(name, pageName string, nameIn []string) (string, []any) {
	if len(nameIn) > 0 {
		seen := map[string]bool{}
		slugs := make([]string, 0, len(nameIn))
		for _, s := range nameIn {
			s = lastPathSegment(strings.TrimSpace(s))
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			slugs = append(slugs, s)
		}
		if len(slugs) == 0 {
			return "post_name IN (%s)", []any{""}
		}
		ph := make([]string, len(slugs))
		vals := make([]any, len(slugs))
		for i, s := range slugs {
			ph[i] = "%s"
			vals[i] = s
		}
		return "post_name IN (" + strings.Join(ph, ",") + ")", vals
	}
	slug := name
	if slug == "" {
		slug = pageName
	}
	if slug == "" {
		return "", nil
	}
	return "post_name = %s", []any{lastPathSegment(slug)}
}

// appendTaxQueryFilters adds REST categories/tags include+exclude as EXISTS clauses (P4.136).
// tax_relation AND|OR joins the clauses (default AND). idCol is "ID" or "p.ID".
func appendTaxQueryFilters(c *Context, where []string, vals []any, args QueryArgs, idCol string) ([]string, []any) {
	if c == nil || c.DB == nil {
		return where, vals
	}
	if idCol == "" {
		idCol = "ID"
	}
	tr := c.DB.TermRelationships()
	tt := c.DB.TermTaxonomy()
	terms := c.DB.Terms()
	termExists := func(taxonomy string, ids []int64, not bool) (string, []any) {
		if len(ids) == 0 {
			return "", nil
		}
		ph := make([]string, len(ids))
		v := make([]any, 0, 1+len(ids))
		v = append(v, taxonomy)
		for i, id := range ids {
			ph[i] = "%d"
			v = append(v, id)
		}
		op := "EXISTS"
		if not {
			op = "NOT EXISTS"
		}
		sql := op + " (SELECT 1 FROM " + tr + " tr_tq INNER JOIN " + tt + " tt_tq ON tt_tq.term_taxonomy_id = tr_tq.term_taxonomy_id INNER JOIN " + terms + " t_tq ON t_tq.term_id = tt_tq.term_id WHERE tr_tq.object_id = " + idCol + " AND tt_tq.taxonomy = %s AND t_tq.term_id IN (" + strings.Join(ph, ",") + "))"
		return sql, v
	}
	var parts []string
	var partVals []any
	add := func(sql string, v []any) {
		if sql == "" {
			return
		}
		parts = append(parts, sql)
		partVals = append(partVals, v...)
	}
	s, v := termExists("category", args.CategoryIn, false)
	add(s, v)
	s, v = termExists("post_tag", args.TagIn, false)
	add(s, v)
	s, v = termExists("category", args.CategoryNotIn, true)
	add(s, v)
	s, v = termExists("post_tag", args.TagNotIn, true)
	add(s, v)
	if len(parts) == 0 {
		return where, vals
	}
	rel := "AND"
	if strings.EqualFold(strings.TrimSpace(args.TaxRelation), "OR") {
		rel = "OR"
	}
	where = append(where, "("+strings.Join(parts, " "+rel+" ")+")")
	vals = append(vals, partVals...)
	return where, vals
}

// appendPostFormatFilter adds REST ?format= tax_query (P4.96).
// idCol is "ID" or "p.ID" depending on whether posts are aliased.
func appendPostFormatFilter(c *Context, where []string, vals []any, args QueryArgs, idCol string) ([]string, []any) {
	if c == nil || c.DB == nil {
		return where, vals
	}
	if len(args.PostFormatIn) == 0 && !args.PostFormatStandard {
		return where, vals
	}
	if idCol == "" {
		idCol = "ID"
	}
	tr := c.DB.TermRelationships()
	tt := c.DB.TermTaxonomy()
	terms := c.DB.Terms()
	var orParts []string
	var orVals []any
	if args.PostFormatStandard {
		orParts = append(orParts,
			"NOT EXISTS (SELECT 1 FROM "+tr+" tr_pf INNER JOIN "+tt+" tt_pf ON tt_pf.term_taxonomy_id = tr_pf.term_taxonomy_id WHERE tr_pf.object_id = "+idCol+" AND tt_pf.taxonomy = %s)")
		orVals = append(orVals, "post_format")
	}
	if len(args.PostFormatIn) > 0 {
		ph := make([]string, len(args.PostFormatIn))
		for i := range args.PostFormatIn {
			ph[i] = "%s"
		}
		orParts = append(orParts,
			"EXISTS (SELECT 1 FROM "+tr+" tr_pf INNER JOIN "+tt+" tt_pf ON tt_pf.term_taxonomy_id = tr_pf.term_taxonomy_id INNER JOIN "+terms+" t_pf ON t_pf.term_id = tt_pf.term_id WHERE tr_pf.object_id = "+idCol+" AND tt_pf.taxonomy = %s AND t_pf.slug IN ("+strings.Join(ph, ",")+"))")
		orVals = append(orVals, "post_format")
		for _, s := range args.PostFormatIn {
			orVals = append(orVals, s)
		}
	}
	where = append(where, "("+strings.Join(orParts, " OR ")+")")
	vals = append(vals, orVals...)
	return where, vals
}

func postInNotInSQL(col string, in, notIn []int64) (string, []any) {
	if col == "" {
		col = "ID"
	}
	if in != nil {
		if len(in) == 0 {
			return col + " IN (%d)", []any{int64(0)}
		}
		ph := make([]string, len(in))
		vals := make([]any, len(in))
		for i, id := range in {
			ph[i] = "%d"
			vals[i] = id
		}
		return col + " IN (" + strings.Join(ph, ",") + ")", vals
	}
	if len(notIn) == 0 {
		return "", nil
	}
	ph := make([]string, len(notIn))
	vals := make([]any, len(notIn))
	for i, id := range notIn {
		ph[i] = "%d"
		vals[i] = id
	}
	return col + " NOT IN (" + strings.Join(ph, ",") + ")", vals
}

// postParentInNotInSQL builds post_parent__in / post_parent__not_in (P4.93).
// Unlike post__in, parent ID 0 (top-level) is a valid include value.
func postParentInNotInSQL(in, notIn []int64) (string, []any) {
	var parts []string
	var vals []any
	if in != nil {
		if len(in) == 0 {
			parts = append(parts, "post_parent IN (%d)")
			vals = append(vals, int64(-1))
		} else {
			ph := make([]string, len(in))
			for i, id := range in {
				ph[i] = "%d"
				vals = append(vals, id)
			}
			parts = append(parts, "post_parent IN ("+strings.Join(ph, ",")+")")
		}
	}
	if len(notIn) > 0 {
		ph := make([]string, len(notIn))
		for i, id := range notIn {
			ph[i] = "%d"
			vals = append(vals, id)
		}
		parts = append(parts, "post_parent NOT IN ("+strings.Join(ph, ",")+")")
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " AND "), vals
}

// GetPost returns the first post matching args, or nil if none.
func GetPost(c *Context, args QueryArgs) (*Post, error) {
	if args.PostsPerPage == 0 {
		args.PostsPerPage = 1
	}
	posts, err := GetPosts(c, args)
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, nil
	}
	return &posts[0], nil
}

// GetPostByID loads a post row by ID with no type/status filter (get_post).
func GetPostByID(c *Context, id int64) (*Post, error) {
	if c == nil || c.DB == nil || id <= 0 {
		return nil, nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT ID, post_author, post_date, post_date_gmt, post_modified, post_modified_gmt, "+
			"post_name, post_status, post_type, post_title, post_content, post_excerpt, "+
			"guid, comment_status, ping_status, post_password, post_parent, menu_order FROM "+
			c.DB.Posts()+" WHERE ID = %d LIMIT 1",
		id,
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
	p := postFromRow(row)
	return &p, nil
}

func postFromRow(row map[string]any) Post {
	return Post{
		ID:            rowInt64(row["ID"]),
		Author:        rowInt64(row["post_author"]),
		Date:          rowString(row["post_date"]),
		DateGMT:       rowString(row["post_date_gmt"]),
		Modified:      rowString(row["post_modified"]),
		ModifiedGMT:   rowString(row["post_modified_gmt"]),
		Name:          rowString(row["post_name"]),
		Status:        rowString(row["post_status"]),
		Type:          rowString(row["post_type"]),
		Title:         rowString(row["post_title"]),
		Content:       rowString(row["post_content"]),
		Excerpt:       rowString(row["post_excerpt"]),
		GUID:          rowString(row["guid"]),
		CommentStatus: rowString(row["comment_status"]),
		PingStatus:    rowString(row["ping_status"]),
		Password:      rowString(row["post_password"]),
		Parent:        rowInt64(row["post_parent"]),
		MenuOrder:     int(rowInt64(row["menu_order"])),
	}
}

func lastPathSegment(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// searchFilterSQL builds the Search LIKE clause (P3.21 / P4.135).
// columns may be post_title, post_content, post_excerpt; invalid names ignored.
// Empty columns → all three (WP default).
func searchFilterSQL(term string, columns []string) (string, []any) {
	term = strings.TrimSpace(term)
	if term == "" {
		return "", nil
	}
	allowed := map[string]bool{
		"post_title":   true,
		"post_content": true,
		"post_excerpt": true,
	}
	var cols []string
	seen := map[string]bool{}
	for _, c := range columns {
		c = strings.TrimSpace(c)
		if !allowed[c] || seen[c] {
			continue
		}
		seen[c] = true
		cols = append(cols, c)
	}
	if len(cols) == 0 {
		cols = []string{"post_title", "post_excerpt", "post_content"}
	}
	like := "%" + db.EscapeLike(term) + "%"
	parts := make([]string, len(cols))
	vals := make([]any, len(cols))
	for i, c := range cols {
		parts[i] = c + " LIKE %s"
		vals[i] = like
	}
	return "(" + strings.Join(parts, " OR ") + ")", vals
}

func rowString(v any) string {
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

func rowInt64(v any) int64 {
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
