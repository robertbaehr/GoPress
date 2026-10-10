package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBlockBindingsRegistry(t *testing.T) {
	r := wp.NewBlockBindingsRegistry()
	if !r.RegisterBlockBindingsSource("core/post-meta", wp.BlockBindingsSource{
		Label:       "Post Meta",
		UsesContext: []string{"postId", "postType"},
	}) {
		t.Fatal("register failed")
	}
	src, ok := r.Get("core/post-meta")
	if !ok || src.Label != "Post Meta" || src.Name != "core/post-meta" {
		t.Fatalf("got=%+v ok=%v", src, ok)
	}
	if !r.UnregisterBlockBindingsSource("core/post-meta") {
		t.Fatal("unregister failed")
	}
	if _, ok := r.Get("core/post-meta"); ok {
		t.Fatal("expected missing after unregister")
	}
}

func TestRegisterDefaultBlockBindingsSources(t *testing.T) {
	r := wp.NewBlockBindingsRegistry()
	wp.RegisterDefaultBlockBindingsSources(r, "twentytwentyfive")
	for _, name := range []string{
		"core/pattern-overrides",
		"core/post-data",
		"core/post-meta",
		"core/term-data",
		"twentytwentyfive/format",
	} {
		if _, ok := r.Get(name); !ok {
			t.Fatalf("missing source %q", name)
		}
	}
	m := r.Map()
	if m["core/post-data"].Label != "Post Data" {
		t.Fatalf("post-data label=%q", m["core/post-data"].Label)
	}
	if m["twentytwentyfive/format"].UsesContext != nil {
		t.Fatalf("tt5 format uses_context should be null-ish, got %v", m["twentytwentyfive/format"].UsesContext)
	}
}

func TestBootRegistersBlockBindings(t *testing.T) {
	c := bootWP(t)
	if c.BlockBindings == nil {
		t.Fatal("nil BlockBindings")
	}
	if _, ok := c.BlockBindings.Get("core/post-meta"); !ok {
		t.Fatal("expected core/post-meta registered at boot")
	}
}
