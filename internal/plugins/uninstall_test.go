package plugins_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestUninstallPluginStub(t *testing.T) {
	c := bootOrSkip(t)
	file := "hello.php"

	var preFired, uninstallFired, hookFired bool
	c.Hooks.AddAction("pre_uninstall_plugin", func(args ...any) {
		preFired = true
		if len(args) < 1 || args[0] != file {
			t.Errorf("pre_uninstall_plugin args=%v", args)
		}
	}, 10, 2)
	c.Hooks.AddAction("uninstall_"+file, func(args ...any) {
		hookFired = true
	}, 10, 0)

	if err := plugins.RegisterUninstallHook(c, file, func(ctx *wp.Context, pluginFile string) {
		uninstallFired = true
		if pluginFile != file {
			t.Errorf("pluginFile=%q", pluginFile)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if !plugins.IsUninstallablePlugin(c, file) {
		t.Fatal("expected uninstallable after RegisterUninstallHook")
	}

	if err := plugins.UninstallPlugin(c, file); err != nil {
		t.Fatal(err)
	}
	if !preFired || !uninstallFired || !hookFired {
		t.Fatalf("pre=%v uninstall=%v hook=%v", preFired, uninstallFired, hookFired)
	}
	if plugins.IsUninstallablePlugin(c, file) {
		t.Fatal("expected not uninstallable after UninstallPlugin")
	}
}
