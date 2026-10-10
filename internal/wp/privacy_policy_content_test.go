package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestDefaultPrivacyPolicyContentGuide(t *testing.T) {
	c := bootWP(t)
	html := wp.GetDefaultPrivacyPolicyContent(c, true, false)
	for _, want := range []string{
		"wp-suggested-text",
		"Who we are",
		"privacy-policy-tutorial",
		"Comments",
		"Cookies",
		"Contact information",
		"Industry regulatory disclosure requirements",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in guide content (%d bytes)", want, len(html))
		}
	}
	if strings.Contains(html, "Suggested text:") {
		t.Fatal("guide (description=true) should not include Suggested text paragraphs")
	}
}

func TestDefaultPrivacyPolicyContentSuggested(t *testing.T) {
	c := bootWP(t)
	html := wp.GetDefaultPrivacyPolicyContent(c, false, false)
	for _, want := range []string{
		"Suggested text:",
		"Our website address is:",
		"Comments",
		"Cookies",
		"Where your data is sent",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in suggested content", want)
		}
	}
	blocks := wp.GetDefaultPrivacyPolicyContent(c, false, true)
	if !strings.Contains(blocks, "<!-- wp:paragraph -->") || !strings.Contains(blocks, "<!-- wp:heading -->") {
		t.Fatalf("expected block comments in %d-byte payload", len(blocks))
	}
}
