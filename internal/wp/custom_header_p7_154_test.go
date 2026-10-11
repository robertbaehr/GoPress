package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.154: Get/SetHeaderImage honors remove-header sentinel.
func TestHeaderImageRemoveSentinelP7154(t *testing.T) {
	c := testBootQuery(t)
	wp.AddThemeSupport(c, "custom-header", map[string]any{"default-image": "https://example.com/default-header.jpg"})
	if err := wp.SetHeaderImage(c, "https://example.com/custom.jpg"); err != nil {
		t.Fatal(err)
	}
	if got := wp.GetHeaderImage(c); got != "https://example.com/custom.jpg" {
		t.Fatalf("custom=%q", got)
	}
	if !wp.HasHeaderImage(c) {
		t.Fatal("expected HasHeaderImage")
	}
	if err := wp.SetHeaderImage(c, ""); err != nil {
		t.Fatal(err)
	}
	// WP: remove-header means no image (default is not re-applied).
	if got := wp.GetHeaderImage(c); got != "" {
		t.Fatalf("remove-header should clear: %q", got)
	}
	if wp.HasHeaderImage(c) {
		t.Fatal("HasHeaderImage after remove")
	}
}

// P7.154: Get/SetHeaderTextColor strip #.
func TestHeaderTextColorP7154(t *testing.T) {
	c := testBootQuery(t)
	if err := wp.SetHeaderTextColor(c, "#abcdef"); err != nil {
		t.Fatal(err)
	}
	if got := wp.GetHeaderTextColor(c); got != "abcdef" {
		t.Fatalf("color=%q", got)
	}
	_ = wp.SetHeaderTextColor(c, "")
}
