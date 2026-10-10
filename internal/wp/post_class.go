package wp

import (
	"strconv"
)

// GetPostClass mirrors get_post_class() for front query loops (P5.19).
func GetPostClass(c *Context, postID int64) []string {
	if c == nil || c.DB == nil || postID <= 0 {
		return nil
	}
	p, err := GetPostByID(c, postID)
	if err != nil || p == nil {
		return nil
	}
	classes := []string{
		"post-" + strconv.FormatInt(p.ID, 10),
		p.Type,
		"type-" + p.Type,
		"status-" + p.Status,
	}
	if p.Type == "post" {
		classes = append(classes, "format-standard")
	}
	// sticky class is added by callers only on the main blog query (is_home).
	classes = append(classes, "hentry")
	classes = append(classes, objectTermClasses(c, postID)...)
	return classes
}

// AppendStickyClass adds "sticky" when postID is in sticky_posts (home query).
func AppendStickyClass(c *Context, postID int64, classes []string) []string {
	if IsSticky(c, postID) {
		return append(classes, "sticky")
	}
	return classes
}

// IsSticky reports whether postID is in the sticky_posts option (is_sticky, P7.67).
func IsSticky(c *Context, postID int64) bool {
	if postID <= 0 {
		return false
	}
	for _, id := range stickyPostIDs(c) {
		if id == postID {
			return true
		}
	}
	return false
}

func objectTermClasses(c *Context, postID int64) []string {
	if c == nil || c.DB == nil || postID <= 0 {
		return nil
	}
	q, args, err := c.DB.Prepare(
		"SELECT t.slug, tt.taxonomy FROM "+c.DB.Terms()+" t "+
			"INNER JOIN "+c.DB.TermTaxonomy()+" tt ON t.term_id = tt.term_id "+
			"INNER JOIN "+c.DB.TermRelationships()+" tr ON tr.term_taxonomy_id = tt.term_taxonomy_id "+
			"WHERE tr.object_id = %d",
		postID,
	)
	if err != nil {
		return nil
	}
	rows, err := c.DB.GetResults(q, args...)
	if err != nil {
		return nil
	}
	var out []string
	for _, row := range rows {
		slug := rowString(row["slug"])
		tax := rowString(row["taxonomy"])
		if slug == "" || tax == "" {
			continue
		}
		prefix := tax
		switch tax {
		case "category":
			prefix = "category"
		case "post_tag":
			prefix = "tag"
		}
		out = append(out, prefix+"-"+slug)
	}
	return out
}
