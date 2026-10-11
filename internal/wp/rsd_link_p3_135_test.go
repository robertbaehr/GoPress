package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.135: rsd_link() emits EditURI with site_url(xmlrpc.php?rsd, rpc).
func TestRsdLinkP3135(t *testing.T) {
	c := &wp.Context{Options: wp.NewOptions(nil, wp.NewObjectCache()), Hooks: wp.NewHooks()}
	c.Options.Cache.Set("siteurl", "http://example.test", "options")

	got := wp.RsdLink(c)
	if !strings.HasPrefix(got, `<link rel="EditURI" type="application/rsd+xml" title="RSD" href="`) {
		t.Fatalf("prefix: %q", got)
	}
	if !strings.Contains(got, "http://example.test/xmlrpc.php?rsd") {
		t.Fatalf("href missing: %q", got)
	}
	if !strings.HasSuffix(got, " />\n") {
		t.Fatalf("suffix: %q", got)
	}

	c.Hooks.AddFilter("site_url", func(value any, _ ...any) any {
		return "https://filtered.test/xmlrpc.php?rsd"
	}, 10, 4)
	got = wp.RsdLink(c)
	if !strings.Contains(got, "https://filtered.test/xmlrpc.php?rsd") {
		t.Fatalf("site_url filter: %q", got)
	}
}

func TestRsdLinkNilContextP3135(t *testing.T) {
	got := wp.RsdLink(nil)
	if !strings.Contains(got, `rel="EditURI"`) || !strings.Contains(got, "xmlrpc.php?rsd") {
		t.Fatalf("nil ctx: %q", got)
	}
}
