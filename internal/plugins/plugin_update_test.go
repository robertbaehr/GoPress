package plugins_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestRejectPluginUpdate(t *testing.T) {
	if err := plugins.RejectPluginUpdate("upgrade-plugin"); err == nil {
		t.Fatal("expected reject for upgrade-plugin")
	}
	if err := plugins.RejectPluginUpdate("update-selected"); err == nil {
		t.Fatal("expected reject for update-selected")
	}
	if err := plugins.RejectPluginUpdate("activate"); err != nil {
		t.Fatalf("activate should be allowed: %v", err)
	}
}

func TestPluginUpdateGoneNoticeRegistered(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	plugins.CheckPluginUpdateGone(c)
	joined := c.CaptureAdminNotices()
	if !strings.Contains(joined, "gopress-plugin-update-gone") || !strings.Contains(joined, "N/A") {
		t.Fatalf("expected plugin-update-gone notice, got %q", joined)
	}
}
