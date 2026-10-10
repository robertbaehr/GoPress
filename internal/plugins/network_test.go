package plugins_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestRejectNetworkActivate(t *testing.T) {
	if err := plugins.RejectNetworkActivate("activate", ""); err != nil {
		t.Fatalf("site activate should be allowed: %v", err)
	}
	if err := plugins.RejectNetworkActivate("activate", "1"); err == nil {
		t.Fatal("expected reject for networkwide=1")
	}
	if err := plugins.RejectNetworkActivate("network-activate", ""); err == nil {
		t.Fatal("expected reject for network-activate action")
	}
}

func TestNetworkActivateGoneNoticeRegistered(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	plugins.CheckNetworkActivateGone(c)
	joined := c.CaptureAdminNotices()
	if !strings.Contains(joined, "gopress-plugin-network-gone") || !strings.Contains(joined, "N/A") {
		t.Fatalf("notice missing: %q", joined)
	}
}
