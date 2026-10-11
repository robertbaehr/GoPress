package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderPageListItemOverlayColorsP5134(t *testing.T) {
	env := &blocks.Env{
		ListPages: func() []blocks.NavLink {
			return []blocks.NavLink{
				{ID: 1, Label: "Parent", URL: "/parent/", Parent: 0},
				{ID: 2, Label: "Child", URL: "/child/", Parent: 1},
			}
		},
	}
	out := blocks.RenderWith(env,
		`<!-- wp:navigation {"overlayTextColor":"vivid-red","overlayBackgroundColor":"black"} --><!-- wp:page-list /--><!-- /wp:navigation -->`)
	if !strings.Contains(out, `has-vivid-red-color`) {
		t.Fatalf("expected overlay text color on nested item: %q", out)
	}
	if !strings.Contains(out, `has-black-background-color`) {
		t.Fatalf("expected overlay background on nested item: %q", out)
	}
	// Top-level item should not get overlay classes.
	if strings.Count(out, `has-vivid-red-color`) != 1 {
		t.Fatalf("overlay color should apply once (nested only): %q", out)
	}
}

func TestRenderPageListItemCustomOverlayP5134(t *testing.T) {
	env := &blocks.Env{
		ListPages: func() []blocks.NavLink {
			return []blocks.NavLink{
				{ID: 1, Label: "Parent", URL: "/parent/", Parent: 0},
				{ID: 2, Label: "Child", URL: "/child/", Parent: 1},
			}
		},
	}
	out := blocks.RenderWith(env,
		`<!-- wp:navigation {"customOverlayTextColor":"#abcdef","customOverlayBackgroundColor":"#112233"} --><!-- wp:page-list /--><!-- /wp:navigation -->`)
	if !strings.Contains(out, `color: #abcdef`) {
		t.Fatalf("expected custom overlay text style: %q", out)
	}
	if !strings.Contains(out, `background-color: #112233`) {
		t.Fatalf("expected custom overlay bg style: %q", out)
	}
}
