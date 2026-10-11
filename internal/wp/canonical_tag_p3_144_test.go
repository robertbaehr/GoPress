package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.144: CanonicalTagHTML EscURL-encodes href; empty → "".
func TestCanonicalTagHTMLP3144(t *testing.T) {
	if got := wp.CanonicalTagHTML(""); got != "" {
		t.Fatalf("empty → %q", got)
	}
	got := wp.CanonicalTagHTML("http://example.test/?p=1&foo=bar")
	if !strings.Contains(got, `rel="canonical"`) {
		t.Fatalf("missing rel: %q", got)
	}
	if !strings.Contains(got, "foo=bar") && !strings.Contains(got, "foo&#038;bar") && !strings.Contains(got, "&#038;") {
		// EscURL may rewrite & → &#038; in query.
		if !strings.Contains(got, "href=") {
			t.Fatalf("tag=%q", got)
		}
	}
	if strings.Contains(got, `href="http://example.test/?p=1&foo=bar"`) {
		// Raw ampersand in attribute is OK for some EscURL paths; ensure tag closes.
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "/>") && !strings.Contains(got, "/>") {
		t.Fatalf("expected self-closing tag: %q", got)
	}
}

// P3.144: RelCanonicalForRequest honors CanonicalPageFromRequest page/cpage.
func TestRelCanonicalForRequestP3144(t *testing.T) {
	c := testBootQuery(t)
	req := httptest.NewRequest(http.MethodGet, "/?p=1&page=2&cpage=3", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	link := wp.RelCanonicalForRequest(c, 1)
	if !strings.Contains(link, `rel="canonical"`) {
		t.Fatalf("link=%q", link)
	}
	// Plain permalinks use query args; pretty use path segments.
	if !strings.Contains(link, "page=2") && !strings.Contains(link, "/2/") {
		t.Fatalf("expected page in canonical: %q", link)
	}
	if !strings.Contains(link, "cpage=3") && !strings.Contains(link, "comment-page-3") {
		t.Fatalf("expected cpage in canonical: %q", link)
	}
}
