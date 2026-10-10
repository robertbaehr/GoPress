package wp_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetCanonicalURLPublishedPost(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	got := wp.GetCanonicalURL(c, 1, 0, 0)
	if !strings.Contains(got, "hello-world") {
		t.Fatalf("canonical=%q", got)
	}
	if !strings.HasSuffix(got, "/") {
		t.Fatalf("expected trailing slash: %q", got)
	}
	link := wp.RelCanonicalLink(c, 1)
	if !strings.Contains(link, `rel="canonical"`) || !strings.Contains(link, got) {
		t.Fatalf("link=%q", link)
	}
}

func TestGetCanonicalURLDraftEmpty(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	if got := wp.GetCanonicalURL(c, 999999991, 0, 0); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestGetCanonicalURLPagePagination(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	base := wp.GetCanonicalURL(c, 1, 0, 0)
	paged := wp.GetCanonicalURL(c, 1, 2, 0)
	if paged == base || !strings.Contains(paged, "/2/") {
		t.Fatalf("paged=%q base=%q", paged, base)
	}
	cpage := wp.GetCanonicalURL(c, 1, 0, 3)
	if !strings.Contains(cpage, "cpage=3") {
		t.Fatalf("cpage=%q", cpage)
	}
}
