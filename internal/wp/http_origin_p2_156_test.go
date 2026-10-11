package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.156: GetAllowedHTTPOrigins includes http/https for home/admin hosts.
func TestGetAllowedHTTPOriginsP2156(t *testing.T) {
	c := testBootQuery(t)
	got := wp.GetAllowedHTTPOrigins(c)
	if len(got) == 0 {
		t.Fatal("expected origins")
	}
	joined := strings.Join(got, ",")
	if !strings.Contains(joined, "http://") || !strings.Contains(joined, "https://") {
		t.Fatalf("origins=%v", got)
	}
	c.Hooks.AddFilter("allowed_http_origins", func(value any, _ ...any) any {
		return []string{"https://allowed.example"}
	}, 10, 1)
	if got := wp.GetAllowedHTTPOrigins(c); len(got) != 1 || got[0] != "https://allowed.example" {
		t.Fatalf("filtered=%v", got)
	}
}

// P2.156: IsAllowedHTTPOrigin gates Origin against allowed list + filter.
func TestIsAllowedHTTPOriginP2156(t *testing.T) {
	c := testBootQuery(t)
	home := strings.TrimRight(wp.HomeURL(c, "", ""), "/")
	u := strings.TrimPrefix(strings.TrimPrefix(home, "https://"), "http://")
	host := u
	if i := strings.IndexByte(u, '/'); i >= 0 {
		host = u[:i]
	}
	allowed := "http://" + host
	if got := wp.IsAllowedHTTPOrigin(c, allowed); got != allowed {
		t.Fatalf("allowed=%q got=%q", allowed, got)
	}
	if got := wp.IsAllowedHTTPOrigin(c, "https://evil.example"); got != "" {
		t.Fatalf("evil=%q", got)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("Origin", allowed)
	c = c.ForRequest(httptest.NewRecorder(), req)
	if got := wp.IsAllowedHTTPOrigin(c, ""); got != allowed {
		t.Fatalf("from header=%q", got)
	}

	c.Hooks.AddFilter("allowed_http_origin", func(value any, _ ...any) any {
		return "https://forced.example"
	}, 10, 2)
	if got := wp.IsAllowedHTTPOrigin(c, "https://evil.example"); got != "https://forced.example" {
		t.Fatalf("filter force=%q", got)
	}
}
