package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetTheGeneratorXHTML(t *testing.T) {
	got := wp.GetTheGenerator("xhtml")
	want := `<meta name="generator" content="WordPress ` + wp.Version + `" />`
	if got != want {
		t.Fatalf("xhtml=%q want %q", got, want)
	}
	if !strings.Contains(got, wp.Version) {
		t.Fatalf("missing version: %q", got)
	}
}

func TestGetTheGeneratorGoPressExport(t *testing.T) {
	got := wp.GetTheGenerator("comment")
	if !strings.Contains(got, "GoPress/"+wp.Version) {
		t.Fatalf("expected GoPress/version comment: %q", got)
	}
}

func TestMetaGeneratorXHTML(t *testing.T) {
	if wp.MetaGeneratorXHTML() != wp.GetTheGenerator("xhtml") {
		t.Fatal("MetaGeneratorXHTML mismatch")
	}
}

func TestGetTheGeneratorRDFAndAliases(t *testing.T) {
	rdf := wp.GetTheGenerator("rdf")
	want := `<admin:generatorAgent rdf:resource="https://wordpress.org/?v=` + wp.Version + `" />`
	if rdf != want {
		t.Fatalf("rdf=%q want %q", rdf, want)
	}
	if wp.GetTheGenerator("rdf_header") != rdf {
		t.Fatal("rdf_header alias mismatch")
	}
	if wp.GetTheGenerator("rss2_head") != wp.GetTheGenerator("rss2") {
		t.Fatal("rss2_head alias mismatch")
	}
	if wp.GetTheGenerator("atom_head") != wp.GetTheGenerator("atom") {
		t.Fatal("atom_head alias mismatch")
	}
}

func TestGetTheGeneratorFiltered(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("get_the_generator_xhtml", func(value any, _ ...any) any {
		return `<meta name="generator" content="Filtered" />`
	}, 10, 1)
	got := wp.GetTheGeneratorFiltered(c, "xhtml")
	if got != `<meta name="generator" content="Filtered" />` {
		t.Fatalf("got %q", got)
	}
}

func TestTheGeneratorFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("the_generator", func(value any, args ...any) any {
		return value.(string) + "<!--x-->"
	}, 10, 2)
	got := wp.TheGenerator(c, "rss2")
	if !strings.Contains(got, "<!--x-->") {
		t.Fatalf("missing the_generator injection: %q", got)
	}
}

func TestMetaGeneratorXHTMLForTypeFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_generator_type", func(value any, _ ...any) any {
		return "html"
	}, 10, 1)
	got := wp.MetaGeneratorXHTMLFor(c)
	want := wp.GetTheGenerator("html")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
