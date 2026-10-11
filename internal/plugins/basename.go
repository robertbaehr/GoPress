package plugins

import (
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// pluginPaths maps normalized logical plugin dirs → real dirs (wp_plugin_paths).
// Used by PluginBasename to resolve symlinked plugin trees.
var (
	pluginPathsMu sync.Mutex
	pluginPaths   = map[string]string{}
)

// ResetPluginPathsForTest clears the symlink remap table (tests only).
func ResetPluginPathsForTest() {
	pluginPathsMu.Lock()
	pluginPaths = map[string]string{}
	pluginPathsMu.Unlock()
}

// RegisterPluginRealpath records a symlink remap for PluginBasename
// (wp_register_plugin_realpath). Returns false when the file's directory sits
// directly under the plugins / mu-plugins directory.
func RegisterPluginRealpath(c *wp.Context, file string) bool {
	file = strings.TrimSpace(file)
	if file == "" {
		return false
	}
	pluginDir := wp.NormalizePath(PluginDir(c))
	muDir := wp.NormalizePath(MUPluginDir(c))

	pluginPath := wp.NormalizePath(path.Dir(wp.NormalizePath(strings.ReplaceAll(file, `\`, `/`))))
	if pluginPath == pluginDir || pluginPath == muDir {
		return false
	}

	resolved, err := filepath.EvalSymlinks(file)
	if err != nil {
		// No usable realpath — still succeed like WP when realpath fails mid-flow,
		// but do not register a remap.
		return true
	}
	pluginRealpath := wp.NormalizePath(path.Dir(wp.NormalizePath(strings.ReplaceAll(resolved, `\`, `/`))))
	if pluginPath != pluginRealpath {
		pluginPathsMu.Lock()
		pluginPaths[pluginPath] = pluginRealpath
		pluginPathsMu.Unlock()
	}
	return true
}

// PluginBasename returns the path of a plugin file relative to WP_PLUGIN_DIR
// or WPMU_PLUGIN_DIR (plugin_basename). Symlink remaps from RegisterPluginRealpath
// are applied longest-realdir-first (arsort parity).
func PluginBasename(c *wp.Context, file string) string {
	file = wp.NormalizePath(file)

	pluginPathsMu.Lock()
	type pair struct{ dir, real string }
	pairs := make([]pair, 0, len(pluginPaths))
	for dir, real := range pluginPaths {
		pairs = append(pairs, pair{dir, real})
	}
	pluginPathsMu.Unlock()
	// arsort by realdir: longer / lexicographically greater first.
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].real == pairs[j].real {
			return pairs[i].dir > pairs[j].dir
		}
		return pairs[i].real > pairs[j].real
	})
	for _, p := range pairs {
		if p.real != "" && strings.HasPrefix(file, p.real) {
			file = p.dir + file[len(p.real):]
		}
	}

	pluginDir := wp.NormalizePath(PluginDir(c))
	muDir := wp.NormalizePath(MUPluginDir(c))
	for _, prefix := range []string{pluginDir, muDir} {
		if prefix == "" {
			continue
		}
		if file == prefix {
			file = ""
			break
		}
		if strings.HasPrefix(file, prefix+"/") {
			file = file[len(prefix)+1:]
			break
		}
	}
	return strings.Trim(file, "/")
}

// PluginDirPath returns the filesystem directory of a plugin file with a
// trailing slash (plugin_dir_path).
func PluginDirPath(file string) string {
	file = strings.ReplaceAll(file, `\`, `/`)
	dir := path.Dir(file)
	if dir == "." {
		return "./"
	}
	if strings.HasSuffix(dir, "/") {
		return dir
	}
	return dir + "/"
}
