package wp

import (
	"html"
	"strings"
	"time"
)

// GetTheGenerator mirrors get_the_generator() (P3.47).
// type is one of: html, xhtml, atom, rss2, rdf, comment, export.
// Front/feed types stay WordPress-shaped for differential parity; comment/export
// identify as GoPress/<Version>.
func GetTheGenerator(genType string) string {
	ver := Version
	esc := html.EscapeString(ver)
	t := strings.ToLower(strings.TrimSpace(genType))
	// WP maps feed head action names when type is empty; accept aliases too (P3.56).
	switch t {
	case "rss2_head", "commentsrss2_head":
		t = "rss2"
	case "rdf_header":
		t = "rdf"
	case "atom_head", "comments_atom_head", "app_head":
		t = "atom"
	case "rss_head", "opml_head":
		t = "comment"
	}
	switch t {
	case "html":
		return `<meta name="generator" content="WordPress ` + esc + `">`
	case "xhtml", "":
		return `<meta name="generator" content="WordPress ` + esc + `" />`
	case "atom":
		return `<generator uri="https://wordpress.org/" version="` + esc + `">WordPress</generator>`
	case "rss2":
		return `<generator>https://wordpress.org/?v=` + esc + `</generator>`
	case "rdf":
		return `<admin:generatorAgent rdf:resource="https://wordpress.org/?v=` + esc + `" />`
	case "comment":
		return `<!-- generator="` + Product + `/` + esc + `" -->`
	case "export":
		return `<!-- generator="` + Product + `/` + esc + `" created="` + time.Now().UTC().Format("2006-01-02 15:04") + `" -->`
	default:
		return `<meta name="generator" content="WordPress ` + esc + `" />`
	}
}

// MetaGeneratorXHTML returns the wp_head meta generator tag (xhtml).
func MetaGeneratorXHTML() string {
	return GetTheGenerator("xhtml")
}
