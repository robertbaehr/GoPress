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
	PostType     string // default "post"
	PostStatus   string // default "publish"
	PostsPerPage int    // default 10; -1 means no LIMIT
	Offset       int    // SQL OFFSET (ignored when PostsPerPage < 0)
	P            int64  // post ID (query var "p")
	Name         string // post slug (query var "name")
	PageName     string // page slug (query var "pagename"); implies post_type=page when PostType unset
	// Search is the `s` query var (title/content LIKE, P3.21).
	Search string
	// Year / MonthNum / Day filter post_date for date archives (P3.22).
	Year     int
	MonthNum int
	Day      int
	// Author is the post_author user ID for author archives (P3.23).
	Author int64
	// Tag is the post_tag slug for tag archives (P3.24).
	Tag string
	// Category is the category slug for category archives (P3.25).
	Category string
	// CategoryIn is category term IDs (WP_Query category__in / REST ?categories=, P4.86).
	CategoryIn []int64
	// TagIn is post_tag term IDs (WP_Query tag__in / REST ?tags=, P4.87).
	TagIn []int64
	// StickyFirst prepends sticky posts (home/blog inherit queries, P5.19).
	StickyFirst bool
	// IgnoreSticky excludes sticky posts from results (non-inherit query loops).
	IgnoreSticky bool
	// PostIn limits results to these IDs (post__in; REST sticky=true, P4.82).
	PostIn []int64
	// PostNotIn excludes these IDs (post__not_in; REST sticky=false, P4.82).
	PostNotIn []int64
	// OrderBy is a WP_Query orderby alias: date (default), title, id, slug (P4.83).
	OrderBy string
	// Order is ASC or DESC (default DESC) (P4.83).
	Order string
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
	status := args.PostStatus
	if status == "" {
		status = "publish"
	}
	var where []string
	var vals []any
	where = append(where, "post_type = %s", "post_status = %s")
	vals = append(vals, postType, status)
	if term := strings.TrimSpace(args.Search); term != "" {
		like := "%" + db.EscapeLike(term) + "%"
		where = append(where, "(post_title LIKE %s OR post_excerpt LIKE %s OR post_content LIKE %s)")
		vals = append(vals, like, like, like)
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
	if args.Author > 0 {
		where = append(where, "post_author = %d")
		vals = append(vals, args.Author)
	}
	if clause, cvals := postInNotInSQL("ID", args.PostIn, args.PostNotIn); clause != "" {
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
		whereSQL := strings.Join(where, " AND ")
		whereSQL = strings.ReplaceAll(whereSQL, "post_type", "p.post_type")
		whereSQL = strings.ReplaceAll(whereSQL, "post_status", "p.post_status")
		whereSQL = strings.ReplaceAll(whereSQL, "post_author", "p.post_author")
		whereSQL = strings.ReplaceAll(whereSQL, "post_title", "p.post_title")
		whereSQL = strings.ReplaceAll(whereSQL, "post_excerpt", "p.post_excerpt")
		whereSQL = strings.ReplaceAll(whereSQL, "post_content", "p.post_content")
		whereSQL = strings.ReplaceAll(whereSQL, "post_date", "p.post_date")
		whereSQL = strings.ReplaceAll(whereSQL, "ID IN", "p.ID IN")
		whereSQL = strings.ReplaceAll(whereSQL, "ID NOT IN", "p.ID NOT IN")
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
	if len(args.TagIn) > 0 {
		from = c.DB.Posts() + " p INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s")
		vals = append(vals, "post_tag")
		clause, cvals := postInNotInSQL("t.term_id", args.TagIn, nil)
		where = append(where, clause)
		vals = append(vals, cvals...)
		whereSQL := strings.Join(where, " AND ")
		whereSQL = strings.ReplaceAll(whereSQL, "post_type", "p.post_type")
		whereSQL = strings.ReplaceAll(whereSQL, "post_status", "p.post_status")
		whereSQL = strings.ReplaceAll(whereSQL, "post_author", "p.post_author")
		whereSQL = strings.ReplaceAll(whereSQL, "post_title", "p.post_title")
		whereSQL = strings.ReplaceAll(whereSQL, "post_excerpt", "p.post_excerpt")
		whereSQL = strings.ReplaceAll(whereSQL, "post_content", "p.post_content")
		whereSQL = strings.ReplaceAll(whereSQL, "post_date", "p.post_date")
		whereSQL = strings.ReplaceAll(whereSQL, "ID IN", "p.ID IN")
		whereSQL = strings.ReplaceAll(whereSQL, "ID NOT IN", "p.ID NOT IN")
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
	if len(args.CategoryIn) > 0 {
		from = c.DB.Posts() + " p INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s")
		vals = append(vals, "category")
		clause, cvals := postInNotInSQL("t.term_id", args.CategoryIn, nil)
		where = append(where, clause)
		vals = append(vals, cvals...)
		whereSQL := strings.Join(where, " AND ")
		whereSQL = strings.ReplaceAll(whereSQL, "post_type", "p.post_type")
		whereSQL = strings.ReplaceAll(whereSQL, "post_status", "p.post_status")
		whereSQL = strings.ReplaceAll(whereSQL, "post_author", "p.post_author")
		whereSQL = strings.ReplaceAll(whereSQL, "post_title", "p.post_title")
		whereSQL = strings.ReplaceAll(whereSQL, "post_excerpt", "p.post_excerpt")
		whereSQL = strings.ReplaceAll(whereSQL, "post_content", "p.post_content")
		whereSQL = strings.ReplaceAll(whereSQL, "post_date", "p.post_date")
		whereSQL = strings.ReplaceAll(whereSQL, "ID IN", "p.ID IN")
		whereSQL = strings.ReplaceAll(whereSQL, "ID NOT IN", "p.ID NOT IN")
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
		whereSQL := strings.Join(where, " AND ")
		whereSQL = strings.ReplaceAll(whereSQL, "post_type", "p.post_type")
		whereSQL = strings.ReplaceAll(whereSQL, "post_status", "p.post_status")
		whereSQL = strings.ReplaceAll(whereSQL, "post_author", "p.post_author")
		whereSQL = strings.ReplaceAll(whereSQL, "post_title", "p.post_title")
		whereSQL = strings.ReplaceAll(whereSQL, "post_excerpt", "p.post_excerpt")
		whereSQL = strings.ReplaceAll(whereSQL, "post_content", "p.post_content")
		whereSQL = strings.ReplaceAll(whereSQL, "post_date", "p.post_date")
		whereSQL = strings.ReplaceAll(whereSQL, "ID IN", "p.ID IN")
		whereSQL = strings.ReplaceAll(whereSQL, "ID NOT IN", "p.ID NOT IN")
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
	status := args.PostStatus
	if status == "" {
		status = "publish"
	}
	limit := args.PostsPerPage
	if limit == 0 {
		limit = 10
	}

	var where []string
	var vals []any

	where = append(where, "post_type = %s")
	vals = append(vals, postType)
	where = append(where, "post_status = %s")
	vals = append(vals, status)

	if args.P > 0 {
		where = append(where, "ID = %d")
		vals = append(vals, args.P)
	}
	slug := args.Name
	if slug == "" {
		slug = args.PageName
	}
	if slug != "" {
		slug = lastPathSegment(slug)
		where = append(where, "post_name = %s")
		vals = append(vals, slug)
	}
	if term := strings.TrimSpace(args.Search); term != "" {
		like := "%" + db.EscapeLike(term) + "%"
		where = append(where, "(post_title LIKE %s OR post_excerpt LIKE %s OR post_content LIKE %s)")
		vals = append(vals, like, like, like)
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
	if args.Author > 0 {
		where = append(where, "post_author = %d")
		vals = append(vals, args.Author)
	}
	if clause, cvals := postInNotInSQL("ID", args.PostIn, args.PostNotIn); clause != "" {
		where = append(where, clause)
		vals = append(vals, cvals...)
	}
	from := c.DB.Posts() + " p"
	idCol := "p.ID"
	if tag := strings.TrimSpace(args.Tag); tag != "" {
		from += " INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s", "t.slug = %s")
		vals = append(vals, "post_tag", tag)
	} else if len(args.TagIn) > 0 {
		from += " INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s")
		vals = append(vals, "post_tag")
		clause, cvals := postInNotInSQL("t.term_id", args.TagIn, nil)
		where = append(where, clause)
		vals = append(vals, cvals...)
	} else if len(args.CategoryIn) > 0 {
		from += " INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s")
		vals = append(vals, "category")
		clause, cvals := postInNotInSQL("t.term_id", args.CategoryIn, nil)
		where = append(where, clause)
		vals = append(vals, cvals...)
	} else if cat := strings.TrimSpace(args.Category); cat != "" {
		cat = lastPathSegment(cat)
		from += " INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.object_id = p.ID" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " tt ON tt.term_taxonomy_id = tr.term_taxonomy_id" +
			" INNER JOIN " + c.DB.Terms() + " t ON t.term_id = tt.term_id"
		where = append(where, "tt.taxonomy = %s", "t.slug = %s")
		vals = append(vals, "category", cat)
	}

	// Qualifying column names when joining taxonomies.
	whereSQL := strings.Join(where, " AND ")
	if strings.Contains(from, " tr ") {
		whereSQL = strings.ReplaceAll(whereSQL, "post_type", "p.post_type")
		whereSQL = strings.ReplaceAll(whereSQL, "post_status", "p.post_status")
		whereSQL = strings.ReplaceAll(whereSQL, "post_author", "p.post_author")
		whereSQL = strings.ReplaceAll(whereSQL, "post_title", "p.post_title")
		whereSQL = strings.ReplaceAll(whereSQL, "post_excerpt", "p.post_excerpt")
		whereSQL = strings.ReplaceAll(whereSQL, "post_content", "p.post_content")
		whereSQL = strings.ReplaceAll(whereSQL, "post_name", "p.post_name")
		whereSQL = strings.ReplaceAll(whereSQL, "post_date", "p.post_date")
		whereSQL = strings.ReplaceAll(whereSQL, "ID =", "p.ID =")
		whereSQL = strings.ReplaceAll(whereSQL, "ID IN", "p.ID IN")
		whereSQL = strings.ReplaceAll(whereSQL, "ID NOT IN", "p.ID NOT IN")
	} else {
		from = c.DB.Posts()
		idCol = "ID"
		_ = idCol
	}

	aliased := strings.Contains(from, " tr ")
	q := "SELECT " + selectPostColumns(aliased) + " FROM " +
		from + " WHERE " + whereSQL +
		" ORDER BY " + orderByClause(args.OrderBy, args.Order, aliased)
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
	if postType == "post" && (args.StickyFirst || args.IgnoreSticky) && args.P == 0 && args.Name == "" && args.PageName == "" && args.Tag == "" && args.Category == "" && len(args.CategoryIn) == 0 && len(args.TagIn) == 0 {
		out = applyStickyOrdering(c, out, args)
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

// orderByClause maps QueryArgs.OrderBy/Order onto SQL (P4.83).
// Supported orderby: date (default), title, id, slug. Order: ASC|DESC (default DESC).
func orderByClause(orderby, order string, aliased bool) string {
	col := orderByPostDate(aliased)
	switch strings.ToLower(strings.TrimSpace(orderby)) {
	case "title":
		if aliased {
			col = "p.post_title"
		} else {
			col = "post_title"
		}
	case "id":
		if aliased {
			col = "p.ID"
		} else {
			col = "ID"
		}
	case "slug", "post_name", "name":
		if aliased {
			col = "p.post_name"
		} else {
			col = "post_name"
		}
	case "date", "post_date", "":
		// default
	default:
		// Unknown aliases fall back to date (safe default).
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(order), "ASC") {
		dir = "ASC"
	}
	return col + " " + dir
}

// postInNotInSQL builds post__in / post__not_in fragments (P4.82).
// Empty PostIn yields "ID IN (0)" so WP_Query returns no rows.
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
