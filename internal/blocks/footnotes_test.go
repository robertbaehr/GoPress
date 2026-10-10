package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestFootnotesSupportsAndAnchor(t *testing.T) {
	env := &blocks.Env{
		LoadFootnotes: func(postID int64) string {
			return `[{"id":"fn-a","content":"Hello note"}]`
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 7})
	out := blocks.RenderWith(env, `<!-- wp:footnotes {"anchor":"notes","className":"is-style-default","fontSize":"small"} /-->`)
	if !strings.Contains(out, `class="wp-block-footnotes`) || !strings.Contains(out, "is-style-default") {
		t.Fatalf("expected wrapper classes: %q", out)
	}
	if !strings.Contains(out, "has-small-font-size") {
		t.Fatalf("expected fontSize support: %q", out)
	}
	if !strings.Contains(out, `id="notes"`) {
		t.Fatalf("expected anchor id: %q", out)
	}
	if !strings.Contains(out, `id="fn-a"`) || !strings.Contains(out, "Hello note") {
		t.Fatalf("expected footnote body: %q", out)
	}
}

func TestFootnotesPasswordRequired(t *testing.T) {
	env := &blocks.Env{
		LoadFootnotes: func(postID int64) string {
			return `[{"id":"fn-a","content":"Secret"}]`
		},
		PasswordRequired: func(postID int64) bool { return true },
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 7, Password: "x"})
	out := blocks.RenderWith(env, `<!-- wp:footnotes /-->`)
	if out != "" {
		t.Fatalf("password-protected footnotes should be empty: %q", out)
	}
}

func TestFootnotesPasswordFieldFallback(t *testing.T) {
	env := &blocks.Env{
		LoadFootnotes: func(postID int64) string {
			return `[{"id":"fn-a","content":"Secret"}]`
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 7, Password: "secret"})
	out := blocks.RenderWith(env, `<!-- wp:footnotes /-->`)
	if out != "" {
		t.Fatalf("password field should hide footnotes: %q", out)
	}
}

func TestFootnotesKsesAndSkipEmptyID(t *testing.T) {
	env := &blocks.Env{
		LoadFootnotes: func(postID int64) string {
			return `[{"id":"","content":"skip me"},{"id":"fn-b","content":"<p onclick=\"x\">Safe</p><script>alert(1)</script>"}]`
		},
		SanitizeHTML: func(html string) string {
			// Stand-in for wp.KsesPost (blocks must not import wp).
			html = strings.ReplaceAll(html, `<script>alert(1)</script>`, "")
			html = strings.ReplaceAll(html, ` onclick="x"`, "")
			return html
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 7})
	out := blocks.RenderWith(env, `<!-- wp:footnotes {"textColor":"contrast","backgroundColor":"base"} /-->`)
	if strings.Contains(out, "skip me") || strings.Contains(out, `id=""`) {
		t.Fatalf("empty id footnote should be skipped: %q", out)
	}
	if !strings.Contains(out, `id="fn-b"`) || !strings.Contains(out, "<p>Safe</p>") {
		t.Fatalf("expected kses-sanitized content: %q", out)
	}
	if strings.Contains(out, "<script>") || strings.Contains(out, "onclick") {
		t.Fatalf("expected script/onclick stripped: %q", out)
	}
	if !strings.Contains(out, "has-contrast-color") || !strings.Contains(out, "has-base-background-color") {
		t.Fatalf("expected color support classes: %q", out)
	}
	if !strings.Contains(out, `aria-label="Jump to footnote reference 1"`) {
		t.Fatalf("expected 1-based index after skip: %q", out)
	}
}
