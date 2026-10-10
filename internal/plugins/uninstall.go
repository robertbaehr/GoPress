package plugins

import (
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// UninstallFunc is invoked when a registered uninstall hook runs (P9.6).
type UninstallFunc func(c *wp.Context, pluginFile string)

// uninstallHooks maps plugin file → Go uninstall callbacks registered in-process.
var uninstallHooks = map[string][]UninstallFunc{}

// RegisterUninstallHook records that pluginFile has an uninstall callback (P9.6).
// Mirrors register_uninstall_hook: persists a marker in the uninstall_plugins
// option and keeps an in-process Go callback list for shims.
func RegisterUninstallHook(c *wp.Context, pluginFile string, fn UninstallFunc) error {
	pluginFile = strings.TrimSpace(pluginFile)
	if pluginFile == "" {
		return fmt.Errorf("plugins: empty plugin file")
	}
	if fn != nil {
		uninstallHooks[pluginFile] = append(uninstallHooks[pluginFile], fn)
	}
	if c == nil || c.Options == nil {
		return nil
	}
	m, err := loadUninstallablePlugins(c)
	if err != nil {
		return err
	}
	if m == nil {
		m = map[any]any{}
	}
	// Store a string marker (PHP stores a callable); enough for is_uninstallable_plugin.
	m[pluginFile] = "gopress"
	return saveUninstallablePlugins(c, m)
}

// IsUninstallablePlugin reports whether uninstall_plugins lists pluginFile (P9.6).
func IsUninstallablePlugin(c *wp.Context, pluginFile string) bool {
	pluginFile = strings.TrimSpace(pluginFile)
	if pluginFile == "" {
		return false
	}
	if len(uninstallHooks[pluginFile]) > 0 {
		return true
	}
	m, err := loadUninstallablePlugins(c)
	if err != nil || m == nil {
		return false
	}
	_, ok := m[pluginFile]
	return ok
}

// UninstallPlugin runs the uninstall stub for pluginFile (P9.6).
//
// Fires pre_uninstall_plugin, invokes registered Go uninstall hooks, fires
// uninstall_{pluginFile}, and clears the uninstall_plugins entry. Does not
// delete plugin files (GoPress does not ship a plugin filesystem installer).
func UninstallPlugin(c *wp.Context, pluginFile string) error {
	pluginFile = strings.TrimSpace(pluginFile)
	if pluginFile == "" {
		return fmt.Errorf("plugins: empty plugin file")
	}
	m, err := loadUninstallablePlugins(c)
	if err != nil {
		return err
	}
	if m == nil {
		m = map[any]any{}
	}

	if c != nil && c.Hooks != nil {
		c.Hooks.DoAction("pre_uninstall_plugin", pluginFile, m)
	}

	for _, fn := range uninstallHooks[pluginFile] {
		if fn != nil {
			fn(c, pluginFile)
		}
	}

	if c != nil && c.Hooks != nil {
		c.Hooks.DoAction("uninstall_"+pluginFile)
	}

	delete(m, pluginFile)
	delete(uninstallHooks, pluginFile)
	if err := saveUninstallablePlugins(c, m); err != nil {
		return err
	}
	// Ensure the plugin is no longer marked active.
	_ = DeactivatePlugin(c, pluginFile)
	return nil
}

func loadUninstallablePlugins(c *wp.Context) (map[any]any, error) {
	if c == nil || c.Options == nil {
		return nil, nil
	}
	raw := c.Options.GetOption("uninstall_plugins", "")
	s, ok := raw.(string)
	if !ok {
		s = fmt.Sprint(raw)
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "a:0:{}" || s == "Array" {
		return map[any]any{}, nil
	}
	v, err := wp.UnserializePHP(s)
	if err != nil {
		return nil, fmt.Errorf("plugins: uninstall_plugins: %w", err)
	}
	m, ok := v.(map[any]any)
	if !ok {
		return map[any]any{}, nil
	}
	return m, nil
}

func saveUninstallablePlugins(c *wp.Context, m map[any]any) error {
	if c == nil || c.Options == nil {
		return fmt.Errorf("plugins: nil options")
	}
	if m == nil {
		m = map[any]any{}
	}
	serialized, err := wp.SerializePHP(m)
	if err != nil {
		return err
	}
	_, err = c.Options.UpdateOption("uninstall_plugins", serialized, "no")
	return err
}
