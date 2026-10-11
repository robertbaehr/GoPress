package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.138: bare hostnames become protocol-relative dns-prefetch hrefs.
func TestResourceHintsBareHostnameDNSPrefetchP3138(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_resource_hints", func(value any, args ...any) any {
		rel, _ := args[0].(string)
		if rel != "dns-prefetch" {
			return value
		}
		return []any{"fonts.gstatic.com", "https://FONTS.gstatic.com/css", "//cdn.Example.com/x.js"}
	}, 10, 2)

	got := wp.ResourceHintsHTML(c)
	if !strings.Contains(got, `href='//fonts.gstatic.com'`) {
		t.Fatalf("bare host: %q", got)
	}
	// Case-insensitive host dedupe: fonts.gstatic.com once.
	if strings.Count(got, "fonts.gstatic.com") != 1 {
		t.Fatalf("expected single fonts.gstatic.com host, got %q", got)
	}
	if !strings.Contains(got, `href='//cdn.Example.com'`) && !strings.Contains(got, `href='//cdn.example.com'`) {
		t.Fatalf("protocol-relative host: %q", got)
	}
	// Default still empty without filter.
	if empty := wp.ResourceHintsHTML(&wp.Context{Hooks: wp.NewHooks()}); empty != "" {
		t.Fatalf("default must stay empty: %q", empty)
	}
}

func TestResourceHintsRejectPathAsBareHostP3138(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_resource_hints", func(value any, args ...any) any {
		rel, _ := args[0].(string)
		if rel != "dns-prefetch" {
			return value
		}
		// Paths / spaces / empty — isBareHostname rejects; EscURL may empty them.
		return []any{"not a host", "/relative/path", "", "  "}
	}, 10, 2)
	got := wp.ResourceHintsHTML(c)
	if strings.Contains(got, "dns-prefetch") {
		t.Fatalf("expected no dns-prefetch for invalid hosts: %q", got)
	}
}
