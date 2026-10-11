package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.43: wp_get_attachment_image_attributes, fetchpriority, decoding validation.
func TestGetAttachmentImageAttrsP843(t *testing.T) {
	c := mediaBoot(t)

	c.Hooks.AddFilter("wp_get_attachment_image_attributes", func(value any, args ...any) any {
		attrs, _ := value.(map[string]string)
		if attrs == nil {
			attrs = map[string]string{}
		}
		attrs["data-testid"] = "attachment-img"
		return attrs
	}, 10, 3)

	html := wp.GetAttachmentImage(c, 5, "thumbnail", false, map[string]string{
		"fetchpriority": "high",
		"decoding":      "sync",
	})
	for _, want := range []string{
		`decoding="sync"`,
		`fetchpriority="high"`,
		`data-testid="attachment-img"`,
		`loading="lazy"`,
		`width="150"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}

	// Invalid decoding omitted; loading=false omitted.
	html = wp.GetAttachmentImage(c, 5, "thumbnail", false, map[string]string{
		"decoding": "bogus",
		"loading":  "false",
	})
	if strings.Contains(html, `decoding=`) {
		t.Fatalf("expected decoding omitted: %q", html)
	}
	if strings.Contains(html, `loading=`) {
		t.Fatalf("expected loading omitted: %q", html)
	}

	// Width override from attr.
	html = wp.GetAttachmentImage(c, 5, "thumbnail", false, map[string]string{
		"width": "99", "height": "88",
	})
	if !strings.Contains(html, `width="99"`) || !strings.Contains(html, `height="88"`) {
		t.Fatalf("override dims: %q", html)
	}
}
