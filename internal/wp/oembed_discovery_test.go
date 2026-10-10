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
	if strings.Contains(out, "format=json") {
		t.Fatalf("json format should omit format query arg: %q", out)
	}
	if wp.OEmbedDiscoveryLinks("", "http://example.test/") != "" {
		t.Fatal("empty home should yield empty")
	}
	if wp.OEmbedDiscoveryLinks("http://example.test", "") != "" {
		t.Fatal("empty permalink should yield empty")
	}
}

func TestGetOEmbedEndpointURL(t *testing.T) {
	jsonURL := wp.GetOEmbedEndpointURL("http://example.test", "http://example.test/p/", "json")
	if jsonURL != "http://example.test/wp-json/oembed/1.0/embed?url=http%3A%2F%2Fexample.test%2Fp%2F" {
		t.Fatalf("json=%q", jsonURL)
	}
	xmlURL := wp.GetOEmbedEndpointURL("http://example.test", "http://example.test/p/", "xml")
	if !strings.Contains(xmlURL, "format=xml") {
		t.Fatalf("xml=%q", xmlURL)
	}
}

func TestOEmbedDiscoveryFilters(t *testing.T) {
	h := wp.NewHooks()
	h.AddFilter("oembed_endpoint_url", func(value any, args ...any) any {
		s, _ := value.(string)
		return s + "&via=filter"
	}, 10, 1)
	h.AddFilter("oembed_discovery_links", func(value any, args ...any) any {
		s, _ := value.(string)
		return s + "<!-- filtered -->\n"
	}, 10, 1)
	out := wp.OEmbedDiscoveryLinksWithHooks(h, "http://example.test", "http://example.test/hello/")
	if !strings.Contains(out, "via=filter") {
		t.Fatalf("expected oembed_endpoint_url filter: %q", out)
	}
	if !strings.Contains(out, "<!-- filtered -->") {
		t.Fatalf("expected oembed_discovery_links filter: %q", out)
	}
}
