package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestThemeSupportsAddCurrentGet(t *testing.T) {
	c := &wp.Context{ThemeFeatures: wp.NewThemeFeatures()}
	if wp.CurrentThemeSupports(c, "post-thumbnails") {
		t.Fatal("expected unsupported")
	}
	wp.AddThemeSupport(c, "post-thumbnails")
	if !wp.CurrentThemeSupports(c, "post-thumbnails") {
		t.Fatal("expected post-thumbnails")
	}
	wp.AddThemeSupport(c, "html5", []any{"search-form", "gallery"})
	if !wp.CurrentThemeSupports(c, "html5", "gallery") {
		t.Fatal("expected html5 gallery")
	}
	if wp.CurrentThemeSupports(c, "html5", "comment-form") {
		t.Fatal("comment-form not registered")
	}
	wp.AddThemeSupport(c, "post-formats", []any{"aside", "quote"})
	if !wp.CurrentThemeSupports(c, "post-formats", "aside") {
		t.Fatal("expected aside")
	}
	wp.RemoveThemeSupport(c, "post-thumbnails")
	if wp.CurrentThemeSupports(c, "post-thumbnails") {
		t.Fatal("removed")
	}
}

func TestRegisterDefaultThemeSupportsBlockTheme(t *testing.T) {
	c := mediaBoot(t)
	if !wp.CurrentThemeSupports(c, "post-thumbnails") {
		t.Fatal("block theme should support post-thumbnails")
	}
	if !wp.CurrentThemeSupports(c, "html5", "gallery") {
		t.Fatal("expected html5 gallery")
	}
	if !wp.CurrentThemeSupports(c, "block-templates") {
		t.Fatal("expected block-templates")
	}
	ts := wp.ThemeSupportsForREST(c)
	if ts["automatic-feed-links"] != true {
		t.Fatalf("automatic-feed-links=%v", ts["automatic-feed-links"])
	}
	if ts["align-wide"] != false {
		t.Fatalf("align-wide=%v", ts["align-wide"])
	}
	html5, ok := ts["html5"].([]any)
	if !ok || len(html5) < 5 {
		t.Fatalf("html5=%v", ts["html5"])
	}
	formats, ok := ts["formats"].([]any)
	if !ok || len(formats) < 2 || formats[0] != "standard" {
		t.Fatalf("formats=%v", ts["formats"])
	}
}
