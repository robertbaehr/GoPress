package plugins

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PluginDependencies is a stub of WP_Plugin_Dependencies (P9.13).
// It reads Requires Plugins headers locally and does not call plugins_api /
// wordpress.org for dependency metadata.
type PluginDependencies struct {
	// Dependencies maps plugin file → sanitized dependency slugs.
	Dependencies map[string][]string
	// DependencySlugs is the unique sorted set of required slugs.
	DependencySlugs []string
	// DependentSlugs maps plugin file → its own slug (dirname / basename).
	DependentSlugs map[string]string
	// InstalledSlugs maps dependency slug → relative plugin filepath when present.
	InstalledSlugs map[string]string
}

var rePluginDepSlug = regexp.MustCompile(`(?i)^[a-z0-9]+(-[a-z0-9]+)*$`)

// SanitizeDependencySlugs parses a comma-separated Requires Plugins header
// (WP_Plugin_Dependencies::sanitize_dependency_slugs subset).
func SanitizeDependencySlugs(slugs string) []string {
	parts := strings.Split(slugs, ",")
	seen := map[string]bool{}
	var out []string
	for _, p := range parts {
		slug := strings.TrimSpace(p)
		if slug == "" || !rePluginDepSlug.MatchString(slug) {
			continue
		}
		slug = strings.ToLower(slug)
		if seen[slug] {
			continue
		}
		seen[slug] = true
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}

// ConvertToSlug mirrors WP_Plugin_Dependencies::convert_to_slug:
// "hello-dolly/hello.php" → "hello-dolly"; "hello.php" → "hello".
func ConvertToSlug(pluginFile string) string {
	pluginFile = strings.TrimSpace(strings.ReplaceAll(pluginFile, "\\", "/"))
	if pluginFile == "" {
		return ""
	}
	if strings.Contains(pluginFile, "/") {
		return filepath.Base(filepath.Dir(pluginFile))
	}
	base := filepath.Base(pluginFile)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// ReadPluginDependencies builds the dependency graph from installed plugin headers (P9.13).
func ReadPluginDependencies(c *wp.Context) (*PluginDependencies, error) {
	installed, err := ListInstalledPlugins(c)
	if err != nil {
		return nil, err
	}
	pd := &PluginDependencies{
		Dependencies:   map[string][]string{},
		DependentSlugs: map[string]string{},
		InstalledSlugs: map[string]string{},
	}
	slugSet := map[string]bool{}
	for _, p := range installed {
		file := p.File
		if file == "" {
			continue
		}
		slug := ConvertToSlug(file)
		pd.InstalledSlugs[slug] = file
		pd.DependentSlugs[file] = slug
		deps := SanitizeDependencySlugs(p.RequiresPlugins)
		if len(deps) == 0 {
			continue
		}
		pd.Dependencies[file] = deps
		for _, d := range deps {
			slugSet[d] = true
		}
	}
	for s := range slugSet {
		pd.DependencySlugs = append(pd.DependencySlugs, s)
	}
	sort.Strings(pd.DependencySlugs)
	return pd, nil
}

// HasDependencies reports whether pluginFile declares Requires Plugins.
func (pd *PluginDependencies) HasDependencies(pluginFile string) bool {
	if pd == nil {
		return false
	}
	_, ok := pd.Dependencies[pluginFile]
	return ok
}

// GetDependencies returns dependency slugs for pluginFile (may be nil).
func (pd *PluginDependencies) GetDependencies(pluginFile string) []string {
	if pd == nil {
		return nil
	}
	return pd.Dependencies[pluginFile]
}

// GetDependents returns plugin files that require the given dependency slug.
func (pd *PluginDependencies) GetDependents(slug string) []string {
	if pd == nil {
		return nil
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	var out []string
	for file, deps := range pd.Dependencies {
		for _, d := range deps {
			if d == slug {
				out = append(out, file)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// GetDependencyFilepath returns the installed filepath for a dependency slug, or "".
func (pd *PluginDependencies) GetDependencyFilepath(slug string) string {
	if pd == nil {
		return ""
	}
	return pd.InstalledSlugs[strings.ToLower(strings.TrimSpace(slug))]
}

// HasUnmetDependencies reports whether any required dependency slug is not installed.
// Active/inactive is not checked in this stub (no remote install path).
func (pd *PluginDependencies) HasUnmetDependencies(pluginFile string) bool {
	if pd == nil {
		return false
	}
	deps := pd.Dependencies[pluginFile]
	for _, d := range deps {
		if pd.GetDependencyFilepath(d) == "" {
			return true
		}
	}
	return false
}
