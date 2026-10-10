package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderDetailsFixture(t *testing.T) {
	out := blocks.Render(readSerialized(t, "core__details"))
	if !strings.Contains(out, `class="wp-block-details"`) || !strings.Contains(out, "Details Summary") {
		t.Fatalf("details fixture: %q", out)
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("details leftover comments: %q", out)
	}
}

func TestRenderDetailsSynthesizeShowContent(t *testing.T) {
	raw := `<!-- wp:details {"showContent":true,"summary":"More","name":"faq","align":"wide"} /-->`
	out := blocks.Render(raw)
	for _, want := range []string{
		`<details`,
		`wp-block-details`,
		`alignwide`,
		`name="faq"`,
		` open`,
		`<summary>More</summary>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderDetailsFetchPriorityLowWhenCollapsed(t *testing.T) {
	raw := `<!-- wp:details {"summary":"Pic"} -->
<details class="wp-block-details"><summary>Pic</summary><img src="/x.jpg" alt="" /></details>
<!-- /wp:details -->`
	out := blocks.Render(raw)
	if !strings.Contains(out, `fetchpriority="low"`) {
		t.Fatalf("expected fetchpriority=low when collapsed: %q", out)
	}

	open := `<!-- wp:details {"showContent":true,"summary":"Pic"} -->
<details class="wp-block-details" open><summary>Pic</summary><img src="/x.jpg" alt="" /></details>
<!-- /wp:details -->`
	outOpen := blocks.Render(open)
	if strings.Contains(outOpen, `fetchpriority="low"`) {
		t.Fatalf("open details should not force fetchpriority=low: %q", outOpen)
	}
}
