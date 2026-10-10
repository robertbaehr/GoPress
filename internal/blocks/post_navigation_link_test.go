package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestPostNavigationLinkDefaults(t *testing.T) {
	env := &blocks.Env{
		AdjacentPost: func(postID int64, previous bool) *blocks.QueryPost {
			if previous {
				return &blocks.QueryPost{ID: 9, Title: "Older", Link: "/older/"}
			}
			return &blocks.QueryPost{ID: 11, Title: "Newer", Link: "/newer/"}
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 10, Title: "Current", Link: "/current/"})
	out := blocks.RenderWith(env, `<!-- wp:post-navigation-link /-->`)
	if !strings.Contains(out, `class="post-navigation-link-next wp-block-post-navigation-link"`) {
		t.Fatalf("wrapper: %q", out)
	}
	if !strings.Contains(out, `<a href="/newer/" rel="next">Next</a>`) {
		t.Fatalf("default next label: %q", out)
	}
	if strings.Contains(out, "aria-hidden") {
		t.Fatalf("default arrow=none should omit arrow: %q", out)
	}

	out = blocks.RenderWith(env, `<!-- wp:post-navigation-link {"type":"previous"} /-->`)
	if !strings.Contains(out, `<a href="/older/" rel="prev">Previous</a>`) {
		t.Fatalf("default previous label: %q", out)
	}
}

func TestPostNavigationLinkShowTitleArrowTextAlign(t *testing.T) {
	env := &blocks.Env{
		AdjacentPost: func(postID int64, previous bool) *blocks.QueryPost {
			return &blocks.QueryPost{ID: 11, Title: "Newer Post", Link: "/newer/"}
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 10, Title: "Current", Link: "/current/"})
	out := blocks.RenderWith(env,
		`<!-- wp:post-navigation-link {"showTitle":true,"arrow":"arrow","textAlign":"right"} /-->`)
	if !strings.Contains(out, `has-text-align-right`) {
		t.Fatalf("textAlign: %q", out)
	}
	if !strings.Contains(out, `<a href="/newer/" rel="next">Newer Post</a>`) {
		t.Fatalf("showTitle link: %q", out)
	}
	if !strings.Contains(out, `wp-block-post-navigation-link__arrow-next is-arrow-arrow`) {
		t.Fatalf("arrow class order: %q", out)
	}
	if !strings.Contains(out, `→`) {
		t.Fatalf("arrow glyph: %q", out)
	}
}

func TestPostNavigationLinkLabelLinkLabel(t *testing.T) {
	env := &blocks.Env{
		AdjacentPost: func(postID int64, previous bool) *blocks.QueryPost {
			return &blocks.QueryPost{ID: 9, Title: "Older Post", Link: "/older/"}
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 10, Title: "Current", Link: "/current/"})
	out := blocks.RenderWith(env,
		`<!-- wp:post-navigation-link {"type":"previous","label":"Previous: ","showTitle":true,"linkLabel":true,"arrow":"chevron"} /-->`)
	if !strings.Contains(out, `post-navigation-link__label`) || !strings.Contains(out, `post-navigation-link__title`) {
		t.Fatalf("label+title spans: %q", out)
	}
	if !strings.Contains(out, `«`) {
		t.Fatalf("chevron: %q", out)
	}
	if !strings.Contains(out, `is-arrow-chevron`) {
		t.Fatalf("chevron class: %q", out)
	}
}

func TestPostNavigationLinkEmptyAdjacent(t *testing.T) {
	env := &blocks.Env{
		AdjacentPost: func(postID int64, previous bool) *blocks.QueryPost {
			return nil
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 10, Title: "Current", Link: "/current/"})
	out := blocks.RenderWith(env, `<!-- wp:post-navigation-link {"showTitle":true,"arrow":"arrow"} /-->`)
	if out != `<div class="post-navigation-link-next wp-block-post-navigation-link"></div>` {
		t.Fatalf("empty adjacent wrapper: %q", out)
	}
}
