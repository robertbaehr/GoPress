package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderQueryTitleSearchPolish(t *testing.T) {
	env := &blocks.Env{SearchQuery: "hello"}
	out := blocks.RenderWith(env, `<!-- wp:query-title {"type":"search","level":2,"textAlign":"center"} /-->`)
	for _, want := range []string{
		`<h2 class="wp-block-query-title has-text-align-center">`,
		"Search results for: \u201Chello\u201D",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}

	out = blocks.RenderWith(env, `<!-- wp:query-title {"type":"search","showSearchTerm":false} /-->`)
	if !strings.Contains(out, "Search results") || strings.Contains(out, "hello") {
		t.Fatalf("showSearchTerm=false: %q", out)
	}

	out = blocks.RenderWith(env, `<!-- wp:query-title {"type":"search","level":0} /-->`)
	if !strings.Contains(out, `<p class="wp-block-query-title">`) {
		t.Fatalf("level=0 expected <p>: %q", out)
	}
}

func TestRenderQueryTitleArchiveShowPrefix(t *testing.T) {
	env := &blocks.Env{ArchiveTitle: "Year: 2024"}
	out := blocks.RenderWith(env, `<!-- wp:query-title {"type":"archive"} /-->`)
	if !strings.Contains(out, `Year: <span>2024</span>`) {
		t.Fatalf("expected prefixed archive title: %q", out)
	}

	out = blocks.RenderWith(env, `<!-- wp:query-title {"type":"archive","showPrefix":false} /-->`)
	if strings.Contains(out, "Year:") || !strings.Contains(out, ">2024<") {
		t.Fatalf("showPrefix=false: %q", out)
	}
}

func TestRenderQueryTitlePostType(t *testing.T) {
	env := &blocks.Env{ArchiveTitle: "Page"}
	out := blocks.RenderWith(env, `<!-- wp:query-title {"type":"post-type"} /-->`)
	if !strings.Contains(out, "Post Type: \u201CPage\u201D") {
		t.Fatalf("expected post-type title: %q", out)
	}
	out = blocks.RenderWith(env, `<!-- wp:query-title {"type":"post-type","showPrefix":false} /-->`)
	if !strings.Contains(out, ">Page<") || strings.Contains(out, "Post Type:") {
		t.Fatalf("showPrefix=false post-type: %q", out)
	}
}
