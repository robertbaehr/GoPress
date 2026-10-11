package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// GetEditTermLink builds get_edit_term_link (P3.161).
// Returns admin term.php URL when the term exists and the current user can
// edit_term. objectType may set post_type; empty uses taxonomy default.
// Applies get_edit_term_link filter. Returns "" when denied/missing (WP null).
func GetEditTermLink(c *Context, termID int64, taxonomy, objectType string) string {
	if c == nil || c.DB == nil || termID <= 0 {
		return ""
	}
	taxonomy = strings.TrimSpace(taxonomy)
	if taxonomy == "" {
		taxonomy = "category"
	}
	q, args, err := c.DB.Prepare(
		"SELECT tt.taxonomy FROM "+c.DB.TermTaxonomy()+" tt WHERE tt.term_id = %d AND tt.taxonomy = %s LIMIT 1",
		termID, taxonomy,
	)
	if err != nil {
		return ""
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		return ""
	}
	tax := strings.TrimSpace(asString(row["taxonomy"]))
	if tax == "" {
		return ""
	}
	if c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "edit_term", termID) {
		return ""
	}
	objectType = strings.TrimSpace(objectType)
	if objectType == "" {
		switch tax {
		case "category", "post_tag":
			objectType = "post"
		default:
			objectType = "post"
		}
	}
	link := AdminURL(c, "term.php?taxonomy="+tax+
		"&tag_ID="+strconv.FormatInt(termID, 10)+
		"&post_type="+objectType)
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("get_edit_term_link", link, termID, tax, objectType); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return link
}

func asString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
