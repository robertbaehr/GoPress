package wp_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestCustomLogoEmpty(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	_ = wp.SetCustomLogoID(c, 0)
	if wp.HasCustomLogo(c) {
		t.Fatal("expected no custom logo")
	}
	if got := wp.GetCustomLogo(c); got != "" {
		t.Fatalf("GetCustomLogo=%q want empty", got)
	}
}

func TestCustomLogoHTML(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	// Fixture attachment 5 is sample-photo-a.jpg.
	if err := wp.SetCustomLogoID(c, 5); err != nil {
		t.Fatalf("SetCustomLogoID: %v", err)
	}
	t.Cleanup(func() { _ = wp.SetCustomLogoID(c, 0) })

	if !wp.HasCustomLogo(c) {
		t.Fatal("expected HasCustomLogo")
	}
	if id := wp.CustomLogoID(c); id != 5 {
		t.Fatalf("CustomLogoID=%d", id)
	}
	html := wp.GetCustomLogo(c)
	for _, want := range []string{
		`class="custom-logo-link"`,
		`rel="home"`,
		`class="custom-logo"`,
		`sample-photo-a`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}

	c.Hooks.AddFilter("get_custom_logo", func(value any, args ...any) any {
		return "FILTERED-LOGO"
	}, 10, 1)
	if got := wp.GetCustomLogo(c); got != "FILTERED-LOGO" {
		t.Fatalf("filter: %q", got)
	}
}

func TestCustomLogoUnlinkHomepage(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	if err := wp.SetCustomLogoID(c, 5); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = wp.SetCustomLogoID(c, 0) })
	wp.AddThemeSupport(c, "custom-logo", map[string]any{"unlink-homepage-logo": true})
	c.Scope = &wp.RequestScope{IsFrontPage: true, Paged: 1}
	html := wp.GetCustomLogo(c)
	if !strings.Contains(html, `<span class="custom-logo-link">`) {
		t.Fatalf("expected unlinked span on front: %q", html)
	}
	if strings.Contains(html, `<a href=`) {
		t.Fatalf("expected no anchor on front with unlink: %q", html)
	}
}

func TestCustomLogoSiteLogoFallback(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	_ = wp.SetCustomLogoID(c, 0)
	_, _ = c.Options.UpdateOption("site_logo", int64(8), "yes")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("site_logo", 0, "yes") })
	if id := wp.CustomLogoID(c); id != 8 {
		t.Fatalf("site_logo fallback CustomLogoID=%d", id)
	}
	if !wp.HasCustomLogo(c) {
		t.Fatal("expected HasCustomLogo via site_logo")
	}
}
