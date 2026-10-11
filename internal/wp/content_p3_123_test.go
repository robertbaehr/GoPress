package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestWpAutopClassicPlainText(t *testing.T) {
	out := wp.WpAutop("Hello\n\nWorld", true)
	if !strings.Contains(out, "<p>Hello</p>") || !strings.Contains(out, "<p>World</p>") {
		t.Fatalf("autop: %q", out)
	}
}

func TestTheContentClassicAutopAndCapitalP(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks(), Shortcodes: wp.NewShortcodes()}
	out := wp.TheContent(c, "Welcome to Wordpress")
	if !strings.Contains(out, "<p>") {
		t.Fatalf("expected autop wrap: %q", out)
	}
	if !strings.Contains(out, "WordPress") || strings.Contains(out, "Wordpress") {
		t.Fatalf("expected capital_P_dangit: %q", out)
	}
}

func TestTheContentSkipsAutopForBlocks(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks(), Shortcodes: wp.NewShortcodes()}
	in := `<!-- wp:paragraph --><p>Hello</p><!-- /wp:paragraph -->`
	out := wp.TheContent(c, in)
	// Should not double-wrap with an outer <p><p>...
	if strings.Contains(out, "<p><p") || strings.HasPrefix(strings.TrimSpace(out), "<p>\n<p") {
		t.Fatalf("block content should not double-wrap: %q", out)
	}
	if !strings.Contains(out, `class="wp-block-paragraph"`) {
		t.Fatalf("expected block paragraph class: %q", out)
	}
}

func TestGetTheContentMoreTag(t *testing.T) {
	raw := "Teaser<!--more-->Rest of post"
	got := wp.GetTheContent(raw, "Read more", false)
	if !strings.Contains(got, "Teaser") || !strings.Contains(got, "more-link") || !strings.Contains(got, "Rest of post") {
		t.Fatalf("more link: %q", got)
	}
	stripped := wp.GetTheContent(raw, "Read more", true)
	if strings.Contains(stripped, "Teaser") || !strings.Contains(stripped, "Rest of post") {
		t.Fatalf("strip teaser: %q", stripped)
	}
}

func TestTheContentDisplayEscapesCDATA(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks(), Shortcodes: wp.NewShortcodes()}
	out := wp.TheContentDisplay(c, "<p>]]></p>")
	if strings.Contains(out, "]]>") || !strings.Contains(out, "]]&gt;") {
		t.Fatalf("CDATA escape: %q", out)
	}
}
