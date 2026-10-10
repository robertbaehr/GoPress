package wp_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSetURLSchemeHTTPS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
	req.TLS = &tls.ConnectionState{}
	c := wp.NewContext(context.TODO())
	c.Req = req
	got := wp.SetURLScheme(c, "http://example.test/path", "")
	if got != "https://example.test/path" {
		t.Fatalf("got %q", got)
	}
	rel := wp.SetURLScheme(c, "http://example.test/foo/bar", "relative")
	if rel != "/foo/bar" {
		t.Fatalf("relative=%q", rel)
	}
}

func TestHomeURLAndSiteURL(t *testing.T) {
	c := bootWP(t)

	home := wp.HomeURL(c, "sample-page/", "http")
	if home == "" || !strings.HasSuffix(home, "sample-page/") {
		t.Fatalf("home=%q", home)
	}
	site := wp.SiteURL(c, "wp-login.php", "http")
	if !strings.Contains(site, "wp-login.php") {
		t.Fatalf("site=%q", site)
	}
	admin := wp.AdminURL(c, "options-general.php")
	if !strings.Contains(admin, "wp-admin/options-general.php") {
		t.Fatalf("admin=%q", admin)
	}
}

func TestHomeURLFilter(t *testing.T) {
	c := bootWP(t)
	c.Hooks.AddFilter("home_url", func(value any, args ...any) any {
		return "http://filtered.test/"
	}, 10, 1)
	if got := wp.HomeURL(c, "", "http"); got != "http://filtered.test/" {
		t.Fatalf("got %q", got)
	}
}
