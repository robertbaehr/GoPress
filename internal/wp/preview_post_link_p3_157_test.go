package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.157: GetPreviewPostLink adds preview=true and honors filter.
func TestGetPreviewPostLinkP3157(t *testing.T) {
	c := testBootQuery(t)
	got := wp.GetPreviewPostLink(c, 1, nil, "")
	if !strings.Contains(got, "preview=true") {
		t.Fatalf("missing preview: %q", got)
	}
	if !strings.Contains(got, "preview=true") || got == "" {
		t.Fatalf("empty/missing: %q", got)
	}

	extra := wp.GetPreviewPostLink(c, 1, map[string]string{"preview_id": "1"}, "")
	if !strings.Contains(extra, "preview_id=1") || !strings.Contains(extra, "preview=true") {
		t.Fatalf("extra args: %q", extra)
	}

	c.Hooks.AddFilter("preview_post_link", func(value any, _ ...any) any {
		return "https://example.test/preview-filtered"
	}, 10, 2)
	if got := wp.GetPreviewPostLink(c, 1, nil, ""); got != "https://example.test/preview-filtered" {
		t.Fatalf("filter=%q", got)
	}
}

// P3.157: custom previewLink seed + missing post.
func TestGetPreviewPostLinkSeedP3157(t *testing.T) {
	c := testBootQuery(t)
	got := wp.GetPreviewPostLink(c, 1, nil, "https://example.test/custom/")
	if !strings.HasPrefix(got, "https://example.test/custom/") || !strings.Contains(got, "preview=true") {
		t.Fatalf("seed=%q", got)
	}
	if got := wp.GetPreviewPostLink(c, 0, nil, ""); got != "" {
		t.Fatalf("zero id=%q", got)
	}
	if got := wp.GetPreviewPostLink(nil, 1, nil, ""); got != "" {
		t.Fatalf("nil ctx=%q", got)
	}
}
