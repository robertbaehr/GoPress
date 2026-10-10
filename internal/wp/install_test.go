package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsInstalledFalseWithoutOptions(t *testing.T) {
	if wp.IsInstalled(nil) {
		t.Fatal("nil context should be not installed")
	}
	c := wp.NewContext(context.TODO())
	if wp.IsInstalled(c) {
		t.Fatal("nil Options should be not installed")
	}
	c.Options = wp.NewOptions(nil, wp.NewObjectCache())
	if wp.IsInstalled(c) {
		t.Fatal("missing blogname should be not installed")
	}
}

func TestIsInstalledTrueWithBlogname(t *testing.T) {
	c := wp.NewContext(context.TODO())
	cache := wp.NewObjectCache()
	cache.Set("alloptions", map[string]string{"blogname": "GoPress Fixture"}, "options")
	c.Options = wp.NewOptions(nil, cache)
	if !wp.IsInstalled(c) {
		t.Fatal("expected installed when blogname present")
	}
}
