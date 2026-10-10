// Template hierarchy helpers (P5.24) — block-theme slug lists matching
// get_home_template / get_page_template / get_single_template / get_singular_template.
package wp

import (
	"fmt"
	"strconv"
	"strings"
)

// HomeTemplateSlugs returns the home/front hierarchy for block themes.
// When isFrontPage is true, front-page is tried before home and index.
func HomeTemplateSlugs(isFrontPage bool) []string {
	if isFrontPage {
		return []string{"front-page", "home", "index"}
	}
	return []string{"home", "index"}
}

// SearchTemplateSlugs returns the search hierarchy for block themes (P3.21).
func SearchTemplateSlugs() []string {
	return []string{"search", "index"}
}

// DateTemplateSlugs returns the date archive hierarchy for block themes (P3.22).
// WP: date → archive → index (block themes typically ship archive.html).
func DateTemplateSlugs() []string {
	return []string{"date", "archive", "index"}
}

// AuthorTemplateSlugs returns the author archive hierarchy for block themes (P3.23).
// WP: author-{nicename}, author-{id}, author, archive, index.
func AuthorTemplateSlugs(nicename string, id int64) []string {
	var slugs []string
	if nicename = strings.TrimSpace(nicename); nicename != "" {
		slugs = append(slugs, "author-"+nicename)
	}
	if id > 0 {
		slugs = append(slugs, "author-"+strconv.FormatInt(id, 10))
	}
	slugs = append(slugs, "author", "archive", "index")
	return slugs
}

// TagTemplateSlugs returns the tag archive hierarchy for block themes (P3.24).
// WP: tag-{slug}, tag-{id}, tag, archive, index.
func TagTemplateSlugs(slug string, id int64) []string {
	var slugs []string
	if slug = strings.TrimSpace(slug); slug != "" {
		slugs = append(slugs, "tag-"+slug)
	}
	if id > 0 {
		slugs = append(slugs, "tag-"+strconv.FormatInt(id, 10))
	}
	slugs = append(slugs, "tag", "archive", "index")
	return slugs
}

// CategoryTemplateSlugs returns the category archive hierarchy for block themes (P3.25).
// WP: category-{slug}, category-{id}, category, archive, index.
func CategoryTemplateSlugs(slug string, id int64) []string {
	var slugs []string
	if slug = strings.TrimSpace(slug); slug != "" {
		slugs = append(slugs, "category-"+slug)
	}
	if id > 0 {
		slugs = append(slugs, "category-"+strconv.FormatInt(id, 10))
	}
	slugs = append(slugs, "category", "archive", "index")
	return slugs
}

// SingularTemplateSlugs returns the singular hierarchy for a post or page.
// Pages: page-{slug}, page-{id}, page, singular, index.
// Posts/CPTs: single-{type}-{slug}, single-{type}, single, singular, index.
func SingularTemplateSlugs(postType, slug string, id int64) []string {
	postType = strings.TrimSpace(postType)
	slug = strings.TrimSpace(slug)
	if postType == "" {
		postType = "post"
	}
	var slugs []string
	if postType == "page" {
		if slug != "" {
			slugs = append(slugs, "page-"+slug)
		}
		if id > 0 {
			slugs = append(slugs, "page-"+strconv.FormatInt(id, 10))
		}
		slugs = append(slugs, "page", "singular", "index")
		return slugs
	}
	if slug != "" {
		slugs = append(slugs, fmt.Sprintf("single-%s-%s", postType, slug))
	}
	slugs = append(slugs, "single-"+postType, "single", "singular", "index")
	return slugs
}
