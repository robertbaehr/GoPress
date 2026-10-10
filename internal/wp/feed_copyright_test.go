package wp_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFeedCopyrightEmptyByDefault(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()
	_, _ = c.Options.DeleteOption("rss_copyright")
	if got := wp.FeedCopyright(c, time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)); got != "" {
		t.Fatalf("expected empty default, got %q", got)
	}
}

func TestFeedCopyrightAutoAndCustom(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()
	t.Cleanup(func() { _, _ = c.Options.DeleteOption("rss_copyright") })

	if _, err := c.Options.UpdateOption("rss_copyright", "auto", "yes"); err != nil {
		t.Fatal(err)
	}
	got := wp.FeedCopyright(c, time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	if got != "Copyright 2024 GoPress Fixture" {
		t.Fatalf("auto: %q", got)
	}

	if _, err := c.Options.UpdateOption("rss_copyright", "© Example Co", "yes"); err != nil {
		t.Fatal(err)
	}
	got = wp.FeedCopyright(c, time.Time{})
	if got != "© Example Co" {
		t.Fatalf("custom: %q", got)
	}
}
