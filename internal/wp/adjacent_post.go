// get_adjacent_post subset (P3.122 / P3.145) — wp-includes/link-template.php.
package wp

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
)

// AdjacentPostResult is a slim adjacent-post row for front/nav consumers.
type AdjacentPostResult struct {
	ID    int64
	Title string
	Name  string
	Date  string
	Type  string
}

// GetAdjacentPost loads the previous/next published post relative to postID
// (get_adjacent_post). When taxonomy is non-empty, restricts to the same term
// (in_same_term). Applies get_{previous|next}_post_{join,where,sort} filters.
func GetAdjacentPost(c *Context, postID int64, previous bool, taxonomy string) (*AdjacentPostResult, error) {
	if c == nil || c.DB == nil || postID <= 0 {
		return nil, nil
	}
	p, err := GetPostByID(c, postID)
	if err != nil || p == nil {
		return nil, err
	}
	adjacent := "previous"
	op, order := "<", "DESC"
	if !previous {
		adjacent = "next"
		op, order = ">", "ASC"
	}
	taxonomy = strings.TrimSpace(taxonomy)
	inSameTerm := taxonomy != ""

	join := ""
	whereExtra := " AND p.post_status = 'publish'"
	var prepArgs []any
	if inSameTerm {
		join = " INNER JOIN " + c.DB.TermRelationships() + " AS tr ON p.ID = tr.object_id" +
			" INNER JOIN " + c.DB.TermTaxonomy() + " AS tt ON tr.term_taxonomy_id = tt.term_taxonomy_id"
		whereExtra += " AND tt.taxonomy = %s"
		prepArgs = append(prepArgs, taxonomy)
		whereExtra += " AND tt.term_taxonomy_id IN (" +
			"SELECT tt2.term_taxonomy_id FROM " + c.DB.TermRelationships() + " tr2 " +
			"INNER JOIN " + c.DB.TermTaxonomy() + " tt2 ON tr2.term_taxonomy_id = tt2.term_taxonomy_id " +
			"WHERE tr2.object_id = %d AND tt2.taxonomy = %s)"
		prepArgs = append(prepArgs, postID, taxonomy)
	}

	if c.Hooks != nil {
		join = asFilterString(c.Hooks.ApplyFilters(
			"get_"+adjacent+"_post_join", join, inSameTerm, "", taxonomy, p,
		), join)
	}

	// ID tie-break for identical dates (WP 6.9+).
	where := "WHERE (p.post_date " + op + " %s OR (p.post_date = %s AND p.ID " + op + " %d))" +
		" AND p.post_type = %s" + whereExtra
	whereArgs := append([]any{p.Date, p.Date, postID, p.Type}, prepArgs...)
	if c.Hooks != nil {
		where = asFilterString(c.Hooks.ApplyFilters(
			"get_"+adjacent+"_post_where", where, inSameTerm, "", taxonomy, p,
		), where)
	}

	sort := "ORDER BY p.post_date " + order + ", p.ID " + order + " LIMIT 1"
	if c.Hooks != nil {
		sort = asFilterString(c.Hooks.ApplyFilters(
			"get_"+adjacent+"_post_sort", sort, p, order,
		), sort)
	}

	sql := "SELECT p.ID, p.post_title, p.post_name, p.post_date, p.post_type FROM " +
		c.DB.Posts() + " AS p " + join + " " + where + " " + sort
	q, args, err := c.DB.Prepare(sql, whereArgs...)
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
	return &AdjacentPostResult{
		ID:    rowInt64(row["ID"]),
		Title: rowString(row["post_title"]),
		Name:  rowString(row["post_name"]),
		Date:  rowString(row["post_date"]),
		Type:  rowString(row["post_type"]),
	}, nil
}

// GetPreviousPost is get_previous_post.
func GetPreviousPost(c *Context, postID int64, taxonomy string) (*AdjacentPostResult, error) {
	return GetAdjacentPost(c, postID, true, taxonomy)
}

// GetNextPost is get_next_post.
func GetNextPost(c *Context, postID int64, taxonomy string) (*AdjacentPostResult, error) {
	return GetAdjacentPost(c, postID, false, taxonomy)
}

func asFilterString(v any, fallback string) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		if v == nil {
			return fallback
		}
		s := fmt.Sprint(v)
		if s == "" || s == "<nil>" {
			return fallback
		}
		return s
	}
}

// AdjacentPostPermalink builds a pretty or ?p= permalink for an adjacent post.
func AdjacentPostPermalink(home string, adj *AdjacentPostResult) string {
	if adj == nil {
		return ""
	}
	home = strings.TrimRight(home, "/")
	link := home + "/?p=" + strconv.FormatInt(adj.ID, 10)
	if adj.Name != "" && adj.Date != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", adj.Date); err == nil {
			link = fmt.Sprintf("%s/%04d/%02d/%02d/%s/", home, t.Year(), int(t.Month()), t.Day(), adj.Name)
		}
	}
	return link
}

// GetAdjacentPostRelLinkHTML mirrors get_adjacent_post_rel_link (P3.145).
// Emits <link rel='prev|next' title='…' href='…' />. Empty when no adjacent post.
// Applies previous_post_rel_link / next_post_rel_link filters.
func GetAdjacentPostRelLinkHTML(c *Context, home string, postID int64, previous bool, taxonomy string) string {
	if postID <= 0 {
		return ""
	}
	// Attachment: previous rel points at parent (WP get_adjacent_post_rel_link).
	if previous && c != nil {
		if p, err := GetPostByID(c, postID); err == nil && p != nil && p.Type == "attachment" && p.Parent > 0 {
			parent, err := GetPostByID(c, p.Parent)
			if err == nil && parent != nil {
				return formatAdjacentPostRelLink(c, home, &AdjacentPostResult{
					ID: parent.ID, Title: parent.Title, Name: parent.Name, Date: parent.Date, Type: parent.Type,
				}, true)
			}
		}
	}
	adj, err := GetAdjacentPost(c, postID, previous, taxonomy)
	if err != nil || adj == nil {
		return ""
	}
	return formatAdjacentPostRelLink(c, home, adj, previous)
}

// AdjacentPostsRelLinkHTML emits prev+next adjacent rel links (adjacent_posts_rel_link, P3.145).
func AdjacentPostsRelLinkHTML(c *Context, home string, postID int64, taxonomy string) string {
	return GetAdjacentPostRelLinkHTML(c, home, postID, true, taxonomy) +
		GetAdjacentPostRelLinkHTML(c, home, postID, false, taxonomy)
}

func formatAdjacentPostRelLink(c *Context, home string, adj *AdjacentPostResult, previous bool) string {
	if adj == nil {
		return ""
	}
	title := strings.TrimSpace(adj.Title)
	if title == "" {
		if previous {
			title = "Previous Post"
		} else {
			title = "Next Post"
		}
	}
	href := EscURL(AdjacentPostPermalink(home, adj))
	rel := "next"
	filter := "next_post_rel_link"
	if previous {
		rel = "prev"
		filter = "previous_post_rel_link"
	}
	link := "<link rel='" + rel + "' title='" + html.EscapeString(title) + "' href='" + html.EscapeString(href) + "' />\n"
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters(filter, link); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return link
}
