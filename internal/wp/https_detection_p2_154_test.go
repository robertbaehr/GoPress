package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.154: http home/site → not using HTTPS.
func TestIsUsingHTTPSFalseOnHTTPFixtureP2154(t *testing.T) {
	c := testBootQuery(t)

	if wp.IsHomeURLUsingHTTPS(c) {
		t.Fatal("fixture home should be http")
	}
	if wp.IsSiteURLUsingHTTPS(c) {
		t.Fatal("fixture siteurl should be http")
	}
	if wp.IsUsingHTTPS(c) {
		t.Fatal("IsUsingHTTPS should be false on http fixture")
	}
	if wp.IsUsingHTTPS(nil) || wp.IsHomeURLUsingHTTPS(nil) || wp.IsSiteURLUsingHTTPS(nil) {
		t.Fatal("nil context → false")
	}
}

// P2.154: both options https → IsUsingHTTPS true.
func TestIsUsingHTTPSTrueWhenBothHTTPSP2154(t *testing.T) {
	c := testBootQuery(t)

	origHome, _ := c.Options.GetOption("home", "").(string)
	origSite, _ := c.Options.GetOption("siteurl", "").(string)
	_, _ = c.Options.UpdateOption("home", "https://example.test", "yes")
	_, _ = c.Options.UpdateOption("siteurl", "https://example.test", "yes")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("home", origHome, "yes")
		_, _ = c.Options.UpdateOption("siteurl", origSite, "yes")
	})

	if !wp.IsHomeURLUsingHTTPS(c) {
		t.Fatal("home https")
	}
	if !wp.IsSiteURLUsingHTTPS(c) {
		t.Fatal("siteurl https")
	}
	if !wp.IsUsingHTTPS(c) {
		t.Fatal("IsUsingHTTPS")
	}
}

// P2.154: site_url filter can flip IsSiteURLUsingHTTPS without changing home.
func TestIsSiteURLUsingHTTPSFilterP2154(t *testing.T) {
	c := testBootQuery(t)

	c.Hooks.AddFilter("site_url", func(value any, _ ...any) any {
		return "https://filtered.example/"
	}, 10, 1)

	if !wp.IsSiteURLUsingHTTPS(c) {
		t.Fatal("filtered site_url should be https")
	}
	// home still http → overall false
	if wp.IsUsingHTTPS(c) {
		t.Fatal("home http should keep IsUsingHTTPS false")
	}
}
