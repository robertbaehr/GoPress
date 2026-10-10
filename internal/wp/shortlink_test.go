package wp_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetShortlinkPost(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	got := wp.GetShortlink(c, 1, "post", 0)
	if !strings.Contains(got, "/?p=1") {
		t.Fatalf("shortlink=%q", got)
	}
	query := wp.GetShortlink(c, 0, "query", 1)
	if query != got {
		t.Fatalf("query context %q != post context %q", query, got)
	}
	if empty := wp.GetShortlink(c, 0, "query", 0); empty != "" {
		t.Fatalf("expected empty without id, got %q", empty)
	}
}
