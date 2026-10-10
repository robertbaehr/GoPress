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
