package wp

import (
	"strconv"
	"strings"
)

// GetPostClass mirrors get_post_class() for front query loops (P5.19 / P3.115).
// Optional cssClass values are space-split extras prepended like WP's $css_class.
// P3.115: password classes, has-post-thumbnail, post_class filter.
func GetPostClass(c *Context, postID int64, cssClass ...string) []string {
	extra := splitPostClasses(cssClass...)
	if c == nil || c.DB == nil || postID <= 0 {
		return extra
	}
	p, err := GetPostByID(c, postID)
	if err != nil || p == nil {
		return extra
	}
	classes := append([]string{}, extra...)
	classes = append(classes,
		"post-"+strconv.FormatInt(p.ID, 10),
		p.Type,
		"type-"+p.Type,
		"status-"+p.Status,
	)
	if p.Type == "post" {
		classes = append(classes, "format-standard")
	}
	// Password classes (post_password_required subset without request → required).
	if strings.TrimSpace(p.Password) != "" {
		if PostPasswordRequired(c, nil, p) {
			classes = append(classes, "post-password-required")
		} else {
			classes = append(classes, "post-password-protected")
		}
	} else if GetPostThumbnailID(c, postID) > 0 {
		// Thumbnail only when not password-gated (WP skips when required).
		classes = append(classes, "has-post-thumbnail")
	}
	// sticky class is added by callers only on the main blog query (is_home).
	classes = append(classes, "hentry")
	classes = append(classes, objectTermClasses(c, postID)...)
	classes = dedupePostClasses(classes)

	if c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("post_class", classes, extra, postID)
		switch t := filtered.(type) {
		case []string:
			classes = t
		case []any:
			classes = make([]string, 0, len(t))
			for _, v := range t {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					classes = append(classes, strings.TrimSpace(s))
				}
			}
		}
		classes = dedupePostClasses(classes)
	}
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

// StickyPostIDs returns sticky_posts option IDs (P4.97 / P7.82).
func StickyPostIDs(c *Context) []int64 {
	return stickyPostIDs(c)
}

// StickPost adds postID to sticky_posts (stick_post, P4.97).
func StickPost(c *Context, postID int64) error {
	if c == nil || c.Options == nil || postID <= 0 {
		return nil
	}
	ids := stickyPostIDs(c)
	for _, id := range ids {
		if id == postID {
			return nil
		}
	}
	ids = append(ids, postID)
	return saveStickyPosts(c, ids)
}

// UnstickPost removes postID from sticky_posts (unstick_post, P4.97).
func UnstickPost(c *Context, postID int64) error {
	if c == nil || c.Options == nil || postID <= 0 {
		return nil
	}
	ids := stickyPostIDs(c)
	out := make([]int64, 0, len(ids))
	found := false
	for _, id := range ids {
		if id == postID {
			found = true
			continue
		}
		out = append(out, id)
	}
	if !found {
		return nil
	}
	return saveStickyPosts(c, out)
}

func saveStickyPosts(c *Context, ids []int64) error {
	// PHP array of ints: a:N:{i:0;i:id;…}
	var b strings.Builder
	b.WriteString("a:")
	b.WriteString(strconv.Itoa(len(ids)))
	b.WriteString(":{")
	for i, id := range ids {
		b.WriteString("i:")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(";i:")
		b.WriteString(strconv.FormatInt(id, 10))
		b.WriteString(";")
	}
	b.WriteString("}")
	_, err := c.Options.UpdateOption("sticky_posts", b.String(), "yes")
	return err
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

func splitPostClasses(cssClass ...string) []string {
	var out []string
	for _, raw := range cssClass {
		for _, p := range strings.Fields(raw) {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func dedupePostClasses(parts []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
