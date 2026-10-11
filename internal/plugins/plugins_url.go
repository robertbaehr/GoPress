package plugins

import (
	"path"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PluginsURL builds a URL within the plugins or mu-plugins directory
// (plugins_url, P9.46). When plugin is under MUPluginDir, the mu-plugins
// base URL is used; otherwise the plugins base. Applies plugins_url filter.
func PluginsURL(c *wp.Context, urlPath, plugin string) string {
	urlPath = wp.NormalizePath(urlPath)
	plugin = wp.NormalizePath(plugin)

	basePath := "wp-content/plugins"
	muDir := wp.NormalizePath(MUPluginDir(c))
	if plugin != "" && muDir != "" && (plugin == muDir || strings.HasPrefix(plugin, muDir+"/")) {
		basePath = "wp-content/mu-plugins"
	}
	url := strings.TrimRight(wp.SiteURL(c, basePath, ""), "/")
	url = wp.SetURLScheme(c, url, "")

	if plugin != "" {
		folder := path.Dir(PluginBasename(c, plugin))
		if folder != "" && folder != "." {
			url += "/" + strings.TrimLeft(folder, "/")
		}
	}
	if urlPath != "" {
		url += "/" + strings.TrimLeft(urlPath, "/")
	}

	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("plugins_url", url, urlPath, plugin); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return url
}

// PluginDirURL returns the URL directory (with trailing slash) for a plugin
// file (plugin_dir_url, P9.46).
func PluginDirURL(c *wp.Context, file string) string {
	url := PluginsURL(c, "", file)
	if url == "" {
		return ""
	}
	if strings.HasSuffix(url, "/") {
		return url
	}
	return url + "/"
}
