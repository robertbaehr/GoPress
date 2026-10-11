package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetAdminNoticeDefaults(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	got := wp.GetAdminNotice(c, "Hello", wp.AdminNoticeArgs{})
	if got != `<div class="notice"><p>Hello</p></div>` {
		t.Fatalf("default notice: %q", got)
	}
}

func TestGetAdminNoticeTypedDismissible(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	got := wp.GetAdminNotice(c, "<strong>Saved</strong>", wp.AdminNoticeArgs{
		Type:        "success",
		Dismissible: true,
		ID:          "message",
	})
	if !strings.Contains(got, `class="notice notice-success is-dismissible"`) {
		t.Fatalf("classes: %q", got)
	}
	if !strings.Contains(got, `id="message"`) {
		t.Fatalf("id: %q", got)
	}
	if !strings.Contains(got, "<p><strong>Saved</strong></p>") {
		t.Fatalf("body: %q", got)
	}
}

func TestGetAdminNoticeNoParagraphWrap(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	got := wp.GetAdminNotice(c, "<p>Err</p>", wp.AdminNoticeArgs{
		Type:              "error",
		ID:                "login_error",
		SkipParagraphWrap: true,
	})
	if got != `<div class="notice notice-error" id="login_error"><p>Err</p></div>` {
		t.Fatalf("no wrap: %q", got)
	}
}

func TestGetAdminNoticeArgsFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_admin_notice_args", func(value any, _ ...any) any {
		a := value.(wp.AdminNoticeArgs)
		a.Type = "warning"
		a.Dismissible = true
		return a
	}, 10, 2)
	got := wp.GetAdminNotice(c, "X", wp.AdminNoticeArgs{Type: "info"})
	if !strings.Contains(got, "notice-warning") || !strings.Contains(got, "is-dismissible") {
		t.Fatalf("args filter: %q", got)
	}
}

func TestGetAdminNoticeMarkupFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_admin_notice_markup", func(value any, _ ...any) any {
		return "<aside>custom</aside>"
	}, 10, 3)
	got := wp.GetAdminNotice(c, "X", wp.AdminNoticeArgs{})
	if got != "<aside>custom</aside>" {
		t.Fatalf("markup filter: %q", got)
	}
}

func TestEmitAdminNoticeBuffersCapture(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddAction("admin_notices", func(args ...any) {
		wp.EmitAdminNotice(c, "From hook", wp.AdminNoticeArgs{Type: "info"})
	}, 10, 0)
	html := c.CaptureAdminNotices()
	if !strings.Contains(html, `class="notice notice-info"`) || !strings.Contains(html, "From hook") {
		t.Fatalf("emit capture: %q", html)
	}
}

func TestFormatSettingsSavedNotice(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	got := wp.FormatSettingsSavedNotice(c)
	if !strings.Contains(got, "notice-success") || !strings.Contains(got, "updated") {
		t.Fatalf("settings saved: %q", got)
	}
	if !strings.Contains(got, "<strong>Settings saved.</strong>") {
		t.Fatalf("message: %q", got)
	}
}
