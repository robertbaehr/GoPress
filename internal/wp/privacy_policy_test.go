package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPrivacyPolicyDraftYieldsEmptyFrontLink(t *testing.T) {
	c := bootWP(t)
	id := wp.PrivacyPolicyPageID(c)
	if id != 3 {
		t.Fatalf("fixture policy page id=%d", id)
	}
	if url := wp.GetPrivacyPolicyURL(c); url != "" {
		t.Fatalf("draft policy should not expose URL, got %q", url)
	}
	if link := wp.GetThePrivacyPolicyLink(c, "[", "]"); link != "" {
		t.Fatalf("draft policy should not emit link, got %q", link)
	}
	sum := wp.PrivacyPolicyAdminSummary(c)
	if !strings.Contains(sum, "Privacy Policy") || !strings.Contains(sum, "draft") {
		t.Fatalf("admin summary=%q", sum)
	}
	if !strings.Contains(sum, "post.php?post=3") {
		t.Fatalf("expected edit link, got %q", sum)
	}
}
