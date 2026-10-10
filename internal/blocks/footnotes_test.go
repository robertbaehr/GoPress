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
