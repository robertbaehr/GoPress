// Package plugins loads enabled GoPress plugins from the active_plugins option (P9.7+).
package plugins

import (
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
	"github.com/robertbaehr/GoPress/plugins/hellodolly"
)

// Plugin is the tiny GoPress plugin interface.
type Plugin interface {
	// File is the WordPress-style plugin basename stored in active_plugins
	// (e.g. "hello.php").
	File() string
	// Activate registers hooks for this plugin on the shared context.
	Activate(c *wp.Context)
}

// registry maps active_plugins entries to built-in Go plugin shims.
var registry = map[string]Plugin{
	"hello.php":                   hellodolly.Plugin{},
	"hellodolly/hellodolly.php":   hellodolly.Plugin{},
	"hello-dolly/hello-dolly.php": hellodolly.Plugin{},
}

// Register adds or replaces a plugin shim for tests / future plugins.
func Register(p Plugin) {
	if p == nil {
		return
	}
	registry[p.File()] = p
}

// LoadActive reads active_plugins and calls Activate on each known shim.
func LoadActive(c *wp.Context) error {
	if c == nil || c.Options == nil {
		return nil
	}
	files, err := ActivePluginFiles(c)
	if err != nil {
		return err
	}
	for _, file := range files {
		p, ok := registry[file]
		if !ok {
			// Unknown PHP-only plugins are ignored in the Go shim.
			continue
		}
		p.Activate(c)
	}
	return nil
}

// ActivePluginFiles returns the plugin file paths listed in active_plugins.
func ActivePluginFiles(c *wp.Context) ([]string, error) {
	if c == nil || c.Options == nil {
		return nil, nil
	}
	raw := c.Options.GetOption("active_plugins", "")
	s, ok := raw.(string)
	if !ok {
		s = fmt.Sprint(raw)
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "a:0:{}" {
		return nil, nil
	}
	v, err := wp.UnserializePHP(s)
	if err != nil {
		return nil, fmt.Errorf("plugins: active_plugins: %w", err)
	}
	m, ok := v.(map[any]any)
	if !ok {
		return nil, nil
	}
	out := make([]string, 0, len(m))
	for i := 0; i < len(m); i++ {
		val, ok := m[int64(i)]
		if !ok {
			val, ok = m[i]
		}
		if !ok {
			continue
		}
		file, _ := val.(string)
		file = strings.TrimSpace(file)
		if file != "" {
			out = append(out, file)
		}
	}
	// Also accept string-keyed maps.
	if len(out) == 0 {
		for _, val := range m {
			file, _ := val.(string)
			file = strings.TrimSpace(file)
			if file != "" {
				out = append(out, file)
			}
		}
	}
	return out, nil
}
