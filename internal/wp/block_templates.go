// Block theme template resolution (P5.5+).
package wp

import (
	"strings"

	"github.com/robertbaehr/GoPress/internal/theme"
)

// ActiveStylesheet returns the active theme stylesheet slug.
func ActiveStylesheet(c *Context) string {
	if c == nil || c.Options == nil {
		return ""
	}
	v, _ := c.Options.GetOption("stylesheet", "").(string)
	v = strings.TrimSpace(v)
	if v == "" {
		return "twentytwentytwo"
	}
	return v
}

// AbsPath returns ABSPATH from config, or "".
func AbsPath(c *Context) string {
	if c == nil || c.Config == nil {
		return ""
	}
	return c.Config.ABSPATH
}

// ResolveBlockTemplate finds block template content for the first matching slug.
// Prefer customized wp_template posts (publish) for the active theme, then
// theme templates/{slug}.html files. Returns content and true when found.
func ResolveBlockTemplate(c *Context, slugs ...string) (string, bool) {
	stylesheet := ActiveStylesheet(c)
	abs := AbsPath(c)
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if content, ok := loadDBBlockTemplate(c, "wp_template", slug, stylesheet); ok {
			return content, true
		}
		if content, ok := theme.LoadTemplateHTML(abs, stylesheet, slug); ok {
			return content, true
		}
	}
	return "", false
}

// ResolveTemplatePart finds wp_template_part content (DB then theme parts/).
func ResolveTemplatePart(c *Context, slug string) (string, bool) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", false
	}
	stylesheet := ActiveStylesheet(c)
	if content, ok := loadDBBlockTemplate(c, "wp_template_part", slug, stylesheet); ok {
		return content, true
	}
	return theme.LoadPartHTML(AbsPath(c), stylesheet, slug)
}

func loadDBBlockTemplate(c *Context, postType, slug, stylesheet string) (string, bool) {
	if c == nil || c.DB == nil {
		return "", false
	}
	post, err := GetPost(c, QueryArgs{
		PostType:     postType,
		Name:         slug,
		PostStatus:   "publish",
		PostsPerPage: 1,
	})
	if err != nil || post == nil {
		return "", false
	}
	if stylesheet != "" && !postBelongsToTheme(c, post.ID, stylesheet) {
		return "", false
	}
	return post.Content, true
}

// postBelongsToTheme reports whether the post is tagged with wp_theme=stylesheet.
// When the post has no wp_theme terms, it is treated as matching (unscoped).
func postBelongsToTheme(c *Context, postID int64, stylesheet string) bool {
	if c == nil || c.DB == nil || postID <= 0 || stylesheet == "" {
		return true
	}
	q := "SELECT t.slug FROM " + c.DB.Terms() + " t " +
		"INNER JOIN " + c.DB.TermTaxonomy() + " tt ON t.term_id = tt.term_id " +
		"INNER JOIN " + c.DB.TermRelationships() + " tr ON tr.term_taxonomy_id = tt.term_taxonomy_id " +
		"WHERE tr.object_id = %d AND tt.taxonomy = %s"
	sqlStr, args, err := c.DB.Prepare(q, postID, "wp_theme")
	if err != nil {
		return true
	}
	rows, err := c.DB.GetResults(sqlStr, args...)
	if err != nil || len(rows) == 0 {
		return true
	}
	for _, row := range rows {
		if rowString(row["slug"]) == stylesheet {
			return true
		}
	}
	return false
}
