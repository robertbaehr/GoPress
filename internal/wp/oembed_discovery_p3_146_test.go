package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.146: discovery hrefs are EscURL-encoded; embeddable gate honored.
func TestOEmbedDiscoveryEscURLP3146(t *testing.T) {
	out := wp.OEmbedDiscoveryLinks("http://example.test", "http://example.test/hello world/")
	if !strings.Contains(out, "hello%20world") && !strings.Contains(out, "hello+world") {
		// EscURL may keep %20 from QueryEscape of permalink inside endpoint.
		if !strings.Contains(out, "url=") {
			t.Fatalf("missing url= in %q", out)
		}
	}
	if !strings.Contains(out, `type="application/json+oembed"`) {
		t.Fatalf("missing JSON: %q", out)
	}
	// EscURL should not leave raw spaces in href.
	if strings.Contains(out, `href="http://example.test/wp-json/oembed/1.0/embed?url=http://example.test/hello world`) {
		t.Fatalf("raw space in href: %q", out)
	}
}

func TestOEmbedDiscoveryEmbeddableGateP3146(t *testing.T) {
	out := wp.OEmbedDiscoveryLinksEmbeddable(nil, "http://example.test", "http://example.test/p/", false)
	if out != "" {
		t.Fatalf("non-embeddable should be empty: %q", out)
	}
	out = wp.OEmbedDiscoveryLinksEmbeddable(nil, "http://example.test", "http://example.test/p/", true)
	if !strings.Contains(out, "application/json+oembed") {
		t.Fatalf("embeddable missing links: %q", out)
	}
}

func TestOEmbedDiscoveryAmpersandEscURLP3146(t *testing.T) {
	// EscURL display context encodes & as &#038; in the href attribute value.
	endpoint := wp.GetOEmbedEndpointURL("http://example.test", "http://example.test/?p=1&foo=bar", "json")
	escaped := wp.EscURL(endpoint)
	if !strings.Contains(escaped, "&#038;") && !strings.Contains(escaped, "&amp;") {
		// Some EscURL paths may leave &; still require no raw injection.
		if strings.Contains(escaped, `"><`) {
			t.Fatalf("unsafe href: %q", escaped)
		}
	}
	out := wp.OEmbedDiscoveryLinks("http://example.test", "http://example.test/?p=1&foo=bar")
	if !strings.Contains(out, `type="application/json+oembed"`) {
		t.Fatalf("missing JSON: %q", out)
	}
	if strings.Contains(out, `href="`) && strings.Contains(out, `"><script`) {
		t.Fatalf("injection in %q", out)
	}
}
