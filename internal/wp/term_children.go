package wp

// GetTermChildren mirrors get_term_children() — all descendant term IDs
// under termID in taxonomy (P4.161). Empty when term has no children.
func GetTermChildren(c *Context, termID int64, taxonomy string) []int64 {
	if c == nil || c.DB == nil || termID <= 0 || taxonomy == "" {
		return nil
	}
	seen := map[int64]bool{}
	var out []int64
	var walk func(parent int64)
	walk = func(parent int64) {
		q, args, err := c.DB.Prepare(
			"SELECT term_id FROM "+c.DB.TermTaxonomy()+" WHERE taxonomy = %s AND parent = %d",
			taxonomy, parent,
		)
		if err != nil {
			return
		}
		rows, err := c.DB.GetResults(q, args...)
		if err != nil {
			return
		}
		for _, row := range rows {
			id := asInt64OrZero(row["term_id"])
			if id <= 0 || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
			walk(id)
		}
	}
	walk(termID)
	return out
}

// ExpandTermIDsWithChildren returns ids plus all descendants when includeChildren
// is true (REST tax_query include_children, P4.161). Dedupes preserving order.
func ExpandTermIDsWithChildren(c *Context, taxonomy string, ids []int64, includeChildren bool) []int64 {
	if len(ids) == 0 {
		return nil
	}
	if !includeChildren {
		return append([]int64(nil), ids...)
	}
	seen := map[int64]bool{}
	var out []int64
	add := func(id int64) {
		if id <= 0 || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range ids {
		add(id)
		for _, child := range GetTermChildren(c, id, taxonomy) {
			add(child)
		}
	}
	return out
}
