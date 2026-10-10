package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFeedTTLEmptyByDefault(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()
	_, _ = c.Options.DeleteOption("rss_ttl")
	if got := wp.FeedTTLMinutes(c); got != "" {
		t.Fatalf("expected empty default, got %q", got)
	}
}

func TestFeedTTLPositiveOption(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()
	t.Cleanup(func() { _, _ = c.Options.DeleteOption("rss_ttl") })

	if _, err := c.Options.UpdateOption("rss_ttl", "60", "yes"); err != nil {
		t.Fatal(err)
	}
	if got := wp.FeedTTLMinutes(c); got != "60" {
		t.Fatalf("got %q", got)
	}
	if _, err := c.Options.UpdateOption("rss_ttl", "0", "yes"); err != nil {
		t.Fatal(err)
	}
	if got := wp.FeedTTLMinutes(c); got != "" {
		t.Fatalf("zero should be empty, got %q", got)
	}
}
