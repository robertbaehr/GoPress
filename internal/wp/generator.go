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
	switch strings.ToLower(strings.TrimSpace(genType)) {
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
