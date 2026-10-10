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
	for _, id := range stickyPostIDs(c) {
		if id == p.ID {
			classes = append(classes, "sticky")
			break
		}
	}
	classes = append(classes, "hentry")
	classes = append(classes, objectTermClasses(c, postID)...)
	return classes
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
