package wp_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPrivacyPolicyGuideHTML(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	wp.ResetPrivacyPolicyContent()
	html := wp.PrivacyPolicyGuideHTML(c)
	for _, want := range []string{
		"WordPress",
		"privacy-text-copy",
		"Suggested text:",
		"Who we are",
		"Copy suggested policy text to clipboard",
		`aria-controls="privacy-settings-accordion-block-1"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}

	wp.AddPrivacyPolicyContent("Acme Plugin", "<p>Acme collects emails.</p>")
	html = wp.PrivacyPolicyGuideHTML(c)
	if !strings.Contains(html, "Acme Plugin") || !strings.Contains(html, "Acme collects emails") {
		t.Fatalf("expected plugin section: %q", html)
	}
	if !strings.Contains(html, "privacy-settings-accordion-block-2") {
		t.Fatalf("expected second accordion block: %q", html)
	}
}
