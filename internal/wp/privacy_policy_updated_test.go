package wp_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPrivacyPolicyUpdatedNotice(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()
	t.Cleanup(func() {
		wp.ResetPrivacyPolicyContent()
		if c.Options != nil {
			_, _ = c.Options.UpdateOption("_wp_suggested_policy_text_has_changed", "not-changed", "no")
		}
	})
	wp.ResetPrivacyPolicyContent()

	pageID := wp.PrivacyPolicyPageID(c)
	if pageID <= 0 {
		t.Fatal("fixture should have a privacy policy page")
	}

	// Seed cache with WordPress default text.
	wp.SeedSuggestedPolicyCache(c)
	if wp.SuggestedPolicyTextHasChanged(c) {
		t.Fatal("expected not-changed after seed")
	}
	if html := wp.PrivacyPolicyUpdatedNoticeHTML(c); html != "" {
		t.Fatalf("unexpected notice: %q", html)
	}

	// Register additional plugin policy text → should flag changed.
	wp.AddPrivacyPolicyContent("Test Plugin", "We collect analytics cookies for Test Plugin.")
	if !wp.TextChangeCheck(c) {
		t.Fatal("expected TextChangeCheck to detect new policy text")
	}
	if !wp.SuggestedPolicyTextHasChanged(c) {
		t.Fatal("expected changed flag")
	}
	html := wp.PrivacyPolicyUpdatedNoticeHTML(c)
	if !strings.Contains(html, "policy-text-updated") {
		t.Fatalf("missing policy-text-updated class: %q", html)
	}
	if !strings.Contains(html, "privacy-policy-guide.php") {
		t.Fatalf("missing guide link: %q", html)
	}
	if !strings.Contains(html, "suggested privacy policy text has changed") {
		t.Fatalf("missing message: %q", html)
	}

	// Reviewing/updating the policy page clears the flag.
	wp.PolicyPageUpdated(c, pageID)
	if wp.SuggestedPolicyTextHasChanged(c) {
		t.Fatal("expected cleared after PolicyPageUpdated")
	}
}
