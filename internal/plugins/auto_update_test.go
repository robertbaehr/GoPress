package plugins_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestRejectAutoUpdate(t *testing.T) {
	if err := plugins.RejectAutoUpdate("activate"); err != nil {
		t.Fatalf("activate should be allowed: %v", err)
	}
	if err := plugins.RejectAutoUpdate("enable-auto-update"); err == nil {
		t.Fatal("expected reject for enable-auto-update")
	}
	if err := plugins.RejectAutoUpdate("disable-auto-update-selected"); err == nil {
		t.Fatal("expected reject for disable-auto-update-selected")
	}
}

func TestAutoUpdateGoneNoticeRegistered(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	plugins.CheckAutoUpdateGone(c)
	joined := c.CaptureAdminNotices()
	if !strings.Contains(joined, "gopress-plugin-auto-update-gone") || !strings.Contains(joined, "N/A") {
		t.Fatalf("notice missing: %q", joined)
	}
}
