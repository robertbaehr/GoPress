package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestOEmbedDiscoveryLinks(t *testing.T) {
	out := wp.OEmbedDiscoveryLinks("http://example.test", "http://example.test/hello-world/")
	if !strings.Contains(out, `type="application/json+oembed"`) {
		t.Fatalf("missing JSON link: %q", out)
	}
	if !strings.Contains(out, `type="text/xml+oembed"`) {
		t.Fatalf("missing XML link: %q", out)
	}
	if !strings.Contains(out, "url=http%3A%2F%2Fexample.test%2Fhello-world%2F") {
		t.Fatalf("missing escaped permalink: %q", out)
	}
	if !strings.Contains(out, "format=xml") {
		t.Fatalf("missing xml format: %q", out)
	}
	if wp.OEmbedDiscoveryLinks("", "http://example.test/") != "" {
		t.Fatal("empty home should yield empty")
	}
	if wp.OEmbedDiscoveryLinks("http://example.test", "") != "" {
		t.Fatal("empty permalink should yield empty")
	}
}
