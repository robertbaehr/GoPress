package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSiteIconRedirectURLDefault(t *testing.T) {
	c := bootWP(t)
	url := wp.SiteIconRedirectURL(c)
	if !strings.Contains(url, wp.DefaultSiteIconPath) {
		t.Fatalf("expected default logo path, got %q", url)
	}
	if wp.HasSiteIcon(c) {
		t.Fatal("fixture should not have a custom site icon")
	}
}
