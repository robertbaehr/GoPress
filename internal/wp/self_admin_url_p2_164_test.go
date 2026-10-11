package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.164: SelfAdminURL builds siteurl/wp-admin/[path] + self_admin_url filter.
func TestSelfAdminURLP2164(t *testing.T) {
	c := testBootQuery(t)
	c.Options.Cache.Set("siteurl", "http://example.test", "options")
	got := wp.SelfAdminURL(c, "", "")
	if !strings.Contains(got, "/wp-admin/") {
		t.Fatalf("base=%q", got)
	}
	got = wp.SelfAdminURL(c, "options-general.php", "")
	if !strings.HasSuffix(got, "/wp-admin/options-general.php") {
		t.Fatalf("path=%q", got)
	}
	c.Hooks.AddFilter("self_admin_url", func(value any, _ ...any) any {
		return "http://cdn.example/admin/"
	}, 10, 3)
	if got := wp.SelfAdminURL(c, "x.php", ""); got != "http://cdn.example/admin/" {
		t.Fatalf("filter=%q", got)
	}
}
