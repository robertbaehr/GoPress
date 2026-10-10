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
