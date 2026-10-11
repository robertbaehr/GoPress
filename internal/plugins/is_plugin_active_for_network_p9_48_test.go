package plugins_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
)

// P9.48: IsPluginActiveForNetwork is always false on single-site GoPress.
func TestIsPluginActiveForNetworkP948(t *testing.T) {
	c := bootOrSkip(t)
	file := "hello.php"
	_ = plugins.ActivatePlugin(c, file)
	t.Cleanup(func() { _ = plugins.DeactivatePlugin(c, file) })

	if plugins.IsPluginActiveForNetwork(c, file) {
		t.Fatal("site-active plugin must not be network-active on single site")
	}
	if plugins.IsPluginActiveForNetwork(c, "no-such-plugin/plugin.php") {
		t.Fatal("missing plugin must not be network-active")
	}
	if plugins.IsPluginActiveForNetwork(nil, file) {
		t.Fatal("nil context must return false")
	}
}
