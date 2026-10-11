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

func TestGetCanonicalURLFilterAndPagedLink(t *testing.T) {
	// P3.101: get_canonical_url filter + RelCanonicalLinkPaged.
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	c.Hooks.AddFilter("get_canonical_url", func(value any, args ...any) any {
		return "http://example.test/canonical-filtered/"
	}, 10, 2)
	if got := wp.GetCanonicalURL(c, 1, 0, 0); got != "http://example.test/canonical-filtered/" {
		t.Fatalf("filtered=%q", got)
	}
	link := wp.RelCanonicalLinkPaged(c, 1, 2, 3)
	if !strings.Contains(link, `rel="canonical"`) || !strings.Contains(link, "canonical-filtered") {
		t.Fatalf("link=%q", link)
	}
}

func TestCanonicalPageFromPrettyPath(t *testing.T) {
	// P3.105: rewrite Scope.Page from /postname/2/ feeds rel=canonical.
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/2024/06/15/hello-world/2/", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	rules, err := c.LoadRewriteRules()
	if err != nil {
		t.Fatalf("rules: %v", err)
	}
	parsed := c.ParseRequest(rules)
	if parsed == nil || parsed.QueryVars["page"] != "2" {
		t.Fatalf("expected page=2 from rewrite, got %#v", parsed)
	}
	page, cpage := wp.CanonicalPageFromRequest(c)
	if page != 2 || cpage != 0 {
		t.Fatalf("page=%d cpage=%d", page, cpage)
	}
	canon := wp.GetCanonicalURL(c, 1, page, cpage)
	if !strings.Contains(canon, "/hello-world/2/") {
		t.Fatalf("canonical=%q", canon)
	}
	link := wp.RelCanonicalLinkPaged(c, 1, page, cpage)
	if !strings.Contains(link, `href="`) || !strings.Contains(link, "/hello-world/2/") {
		t.Fatalf("link=%q", link)
	}
}
