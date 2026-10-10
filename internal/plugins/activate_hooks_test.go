package plugins_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
)

func TestActivateDeactivatePluginHooks(t *testing.T) {
	c := bootOrSkip(t)
	_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")
	})

	var got []string
	c.Hooks.AddAction("activate_plugin", func(args ...any) {
		got = append(got, "activate_plugin")
		if len(args) < 1 || args[0] != "hello.php" {
			t.Fatalf("activate_plugin args=%v", args)
		}
	}, 10, 2)
	c.Hooks.AddAction("activate_hello.php", func(args ...any) {
		got = append(got, "activate_hello.php")
	}, 10, 1)
	c.Hooks.AddAction("activated_plugin", func(args ...any) {
		got = append(got, "activated_plugin")
	}, 10, 2)
	c.Hooks.AddAction("deactivate_plugin", func(args ...any) {
		got = append(got, "deactivate_plugin")
	}, 10, 2)
	c.Hooks.AddAction("deactivate_hello.php", func(args ...any) {
		got = append(got, "deactivate_hello.php")
	}, 10, 1)
	c.Hooks.AddAction("deactivated_plugin", func(args ...any) {
		got = append(got, "deactivated_plugin")
	}, 10, 2)

	if err := plugins.ActivatePlugin(c, "hello.php"); err != nil {
		t.Fatal(err)
	}
	if !plugins.IsPluginActive(c, "hello.php") {
		t.Fatal("expected active")
	}
	if err := plugins.DeactivatePlugin(c, "hello.php"); err != nil {
		t.Fatal(err)
	}
	if plugins.IsPluginActive(c, "hello.php") {
		t.Fatal("expected inactive")
	}
	want := []string{
		"activate_plugin", "activate_hello.php", "activated_plugin",
		"deactivate_plugin", "deactivate_hello.php", "deactivated_plugin",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestActivePluginsFilter(t *testing.T) {
	c := bootOrSkip(t)
	_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")
	})
	c.Hooks.AddFilter("active_plugins", func(value any, args ...any) any {
		return []string{"filtered.php"}
	}, 10, 1)
	files, err := plugins.ActivePluginFiles(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "filtered.php" {
		t.Fatalf("files=%v", files)
	}
}
