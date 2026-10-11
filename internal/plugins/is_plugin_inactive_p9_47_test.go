package plugins_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
)

// P9.47: IsPluginInactive is the inverse of IsPluginActive.
func TestIsPluginInactiveP947(t *testing.T) {
	c := bootOrSkip(t)
	file := "hello.php"
	_ = plugins.ActivatePlugin(c, file)
	t.Cleanup(func() { _ = plugins.DeactivatePlugin(c, file) })

	if plugins.IsPluginInactive(c, file) {
		t.Fatal("active plugin should not be inactive")
	}
	_ = plugins.DeactivatePlugin(c, file)
	if !plugins.IsPluginInactive(c, file) {
		t.Fatal("deactivated plugin should be inactive")
	}
	if !plugins.IsPluginInactive(c, "no-such-plugin/plugin.php") {
		t.Fatal("missing plugin should be inactive")
	}
}
