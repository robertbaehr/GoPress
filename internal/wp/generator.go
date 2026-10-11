package wp

import (
	"html"
	"strings"
	"time"
)

// normalizeGeneratorType maps feed-head action aliases to generator types (P3.56 / P3.121).
func normalizeGeneratorType(genType string) string {
	t := strings.ToLower(strings.TrimSpace(genType))
	switch t {
	case "rss2_head", "commentsrss2_head":
		return "rss2"
	case "rdf_header":
		return "rdf"
	case "atom_head", "comments_atom_head", "app_head":
		return "atom"
	case "rss_head", "opml_head":
		return "comment"
	default:
		return t
	}
}

// GetTheGenerator mirrors get_the_generator() without filters (P3.47).
// type is one of: html, xhtml, atom, rss2, rdf, comment, export.
// Front/feed types stay WordPress-shaped for differential parity; comment/export
// identify as GoPress/<Version>.
func GetTheGenerator(genType string) string {
	return getTheGeneratorRaw(normalizeGeneratorType(genType))
}

func getTheGeneratorRaw(t string) string {
	ver := Version
	esc := html.EscapeString(ver)
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

// GetTheGeneratorFiltered mirrors get_the_generator() with get_the_generator_{$type} (P3.121).
func GetTheGeneratorFiltered(c *Context, genType string) string {
	t := normalizeGeneratorType(genType)
	if t == "" {
		t = "xhtml"
	}
	gen := getTheGeneratorRaw(t)
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("get_the_generator_"+t, gen, t)
		if s, ok := filtered.(string); ok {
			gen = s
		}
	}
	return gen
}

// TheGenerator mirrors the_generator(): get_the_generator + the_generator filter (P3.121).
func TheGenerator(c *Context, genType string) string {
	t := normalizeGeneratorType(genType)
	if t == "" {
		t = "xhtml"
	}
	gen := GetTheGeneratorFiltered(c, t)
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("the_generator", gen, t)
		if s, ok := filtered.(string); ok {
			gen = s
		}
	}
	return gen
}

// MetaGeneratorXHTML returns the wp_head meta generator tag (xhtml) without hooks.
func MetaGeneratorXHTML() string {
	return GetTheGenerator("xhtml")
}

// MetaGeneratorXHTMLFor returns wp_head generator via wp_generator_type + filters (P3.121).
func MetaGeneratorXHTMLFor(c *Context) string {
	genType := "xhtml"
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("wp_generator_type", genType)
		if s, ok := filtered.(string); ok && strings.TrimSpace(s) != "" {
			genType = strings.TrimSpace(s)
		}
	}
	return TheGenerator(c, genType)
}
