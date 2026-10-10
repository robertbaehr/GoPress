// Package plugins loads enabled GoPress plugins from the active_plugins option (P9.7+)
// and must-use plugins from wp-content/mu-plugins (P9.1).
package plugins

import (
	"fmt"
	"os"
	"path/filepath"
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

// registry maps active_plugins / mu-plugin entries to built-in Go plugin shims.
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

// LoadActive loads must-use plugins then active_plugins shims.
func LoadActive(c *wp.Context) error {
	if err := LoadMUPlugins(c); err != nil {
		return err
	}
	if c == nil || c.Options == nil {
		return nil
	}
	files, err := ActivePluginFiles(c)
	if err != nil {
		return err
	}
	for _, file := range files {
		activateRegistered(c, file)
	}
	return nil
}

// LoadMUPlugins scans wp-content/mu-plugins for .php drop-ins and activates
// any matching registered Go shims (WordPress must-use plugin loader subset).
func LoadMUPlugins(c *wp.Context) error {
	if c == nil {
		return nil
	}
	dir := MUPluginDir(c)
	if dir == "" {
		return nil
	}
	files, err := ScanMUPluginFiles(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, file := range files {
		activateRegistered(c, file)
	}
	return nil
}

// MUPluginDir returns ABSPATH/wp-content/mu-plugins (empty when unknown).
func MUPluginDir(c *wp.Context) string {
	if c == nil || c.Config == nil {
		return ""
	}
	root := strings.TrimSpace(c.Config.ABSPATH)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "wp-content", "mu-plugins")
}

// ScanMUPluginFiles lists mu-plugin relative paths the same way WordPress does:
// top-level *.php files and one-level */*.php plugin files.
func ScanMUPluginFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ent := range entries {
		name := ent.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !ent.IsDir() {
			if strings.HasSuffix(strings.ToLower(name), ".php") {
				out = append(out, name)
			}
			continue
		}
		sub := filepath.Join(dir, name)
		subs, err := os.ReadDir(sub)
		if err != nil {
			continue
		}
		for _, se := range subs {
			if se.IsDir() {
				continue
			}
			sn := se.Name()
			if strings.HasSuffix(strings.ToLower(sn), ".php") {
				out = append(out, filepath.ToSlash(filepath.Join(name, sn)))
			}
		}
	}
	return out, nil
}

func activateRegistered(c *wp.Context, file string) {
	p, ok := registry[file]
	if !ok {
		// Unknown PHP-only plugins are ignored in the Go shim.
		return
	}
	p.Activate(c)
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
