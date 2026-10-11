package wp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.129: pretty comment-page path feeds CanonicalPageFromRequest + #comments URL.
func TestCanonicalCommentPagePrettyP3129(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/2024/06/15/hello-world/comment-page-2/", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	_, cpage := wp.CanonicalPageFromRequest(c)
	if cpage != 2 {
		t.Fatalf("cpage=%d want 2", cpage)
	}
	canon := wp.GetCanonicalURL(c, 1, 0, cpage)
	want := "comment-page-2/#comments"
	if !strings.Contains(canon, want) {
		t.Fatalf("canonical=%q want substring %q", canon, want)
	}
	link := wp.RelCanonicalLinkPaged(c, 1, 0, cpage)
	if !strings.Contains(link, want) {
		t.Fatalf("link=%q", link)
	}
}

func TestCanonicalInheritOrphanAttachmentEmptyP3129(t *testing.T) {
	// Attachments with inherit + no published parent have no canonical.
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	// Fixture attachments 5–9 have post_parent=0 and status inherit.
	if got := wp.GetCanonicalURL(c, 5, 0, 0); got != "" {
		t.Fatalf("orphan attachment canonical=%q want empty", got)
	}
}
