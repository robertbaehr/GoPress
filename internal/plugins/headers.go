package plugins

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PluginData is the metadata from a plugin file header (get_plugin_data subset).
type PluginData struct {
	Name            string
	PluginURI       string
	Version         string
	Description     string
	Author          string
	AuthorURI       string
	TextDomain      string
	DomainPath      string
	Network         bool
	RequiresWP      string
	RequiresPHP     string
	UpdateURI       string
	RequiresPlugins string
	Title           string
	AuthorName      string
	File            string // relative plugin file path when known
}

var pluginHeaderMap = []struct {
	field string
	label string
}{
	{"Name", "Plugin Name"},
	{"PluginURI", "Plugin URI"},
	{"Version", "Version"},
	{"Description", "Description"},
	{"Author", "Author"},
	{"AuthorURI", "Author URI"},
	{"TextDomain", "Text Domain"},
	{"DomainPath", "Domain Path"},
	{"Network", "Network"},
	{"RequiresWP", "Requires at least"},
	{"RequiresPHP", "Requires PHP"},
	{"UpdateURI", "Update URI"},
	{"RequiresPlugins", "Requires Plugins"},
}

// ListMUPlugins returns must-use plugin headers like get_mu_plugins() (P9.10).
// Only top-level *.php files under wp-content/mu-plugins are included; when a
// file has no Plugin Name header, Name defaults to the basename.
func ListMUPlugins(c *wp.Context) ([]PluginData, error) {
	dir := MUPluginDir(c)
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []PluginData
	for _, ent := range entries {
		name := ent.Name()
		if ent.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ".php") {
			continue
		}
		abs := filepath.Join(dir, name)
		data, err := GetPluginData(abs)
		if err != nil {
			continue
		}
		if data.Name == "" {
			data.Name = name
			data.Title = name
		}
		data.File = name
		out = append(out, data)
	}
	return out, nil
}

// GetPluginData parses WordPress plugin headers from the first 8 KiB of a .php file (P9.2).
func GetPluginData(pluginFile string) (PluginData, error) {
	f, err := os.Open(pluginFile)
	if err != nil {
		return PluginData{}, err
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return PluginData{}, err
	}
	raw := buf[:n]
	headers := parseFileHeaders(raw, pluginHeaderMap)

	data := PluginData{
		Name:            headers["Name"],
		PluginURI:       headers["PluginURI"],
		Version:         headers["Version"],
		Description:     headers["Description"],
		Author:          headers["Author"],
		AuthorURI:       headers["AuthorURI"],
		TextDomain:      headers["TextDomain"],
		DomainPath:      headers["DomainPath"],
		Network:         strings.EqualFold(headers["Network"], "true"),
		RequiresWP:      headers["RequiresWP"],
		RequiresPHP:     headers["RequiresPHP"],
		UpdateURI:       headers["UpdateURI"],
		RequiresPlugins: headers["RequiresPlugins"],
	}
	data.Title = data.Name
	data.AuthorName = data.Author
	return data, nil
}

// PluginDir returns ABSPATH/wp-content/plugins.
func PluginDir(c *wp.Context) string {
	if c == nil || c.Config == nil {
		return ""
	}
	root := strings.TrimSpace(c.Config.ABSPATH)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "wp-content", "plugins")
}

// ListInstalledPlugins scans wp-content/plugins for plugin headers (P9.2).
func ListInstalledPlugins(c *wp.Context) ([]PluginData, error) {
	dir := PluginDir(c)
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []PluginData
	for _, ent := range entries {
		name := ent.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		var rel string
		var abs string
		if !ent.IsDir() {
			if !strings.HasSuffix(strings.ToLower(name), ".php") {
				continue
			}
			rel = name
			abs = filepath.Join(dir, name)
		} else {
			// One-level plugin directory: first *.php with a Plugin Name header.
			sub := filepath.Join(dir, name)
			subs, err := os.ReadDir(sub)
			if err != nil {
				continue
			}
			for _, se := range subs {
				if se.IsDir() || !strings.HasSuffix(strings.ToLower(se.Name()), ".php") {
					continue
				}
				cand := filepath.Join(sub, se.Name())
				data, err := GetPluginData(cand)
				if err != nil || data.Name == "" {
					continue
				}
				rel = filepath.ToSlash(filepath.Join(name, se.Name()))
				abs = cand
				break
			}
			if abs == "" {
				continue
			}
		}
		data, err := GetPluginData(abs)
		if err != nil || data.Name == "" {
			continue
		}
		data.File = rel
		out = append(out, data)
	}
	return out, nil
}

func parseFileHeaders(raw []byte, headers []struct{ field, label string }) map[string]string {
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		out[h.field] = ""
	}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		for _, h := range headers {
			prefix := h.label + ":"
			if strings.HasPrefix(line, prefix) {
				out[h.field] = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
	}
	return out
}
