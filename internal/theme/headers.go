package theme

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// ThemeData is metadata from a theme's style.css header (get_theme_data subset, P7.34).
type ThemeData struct {
	Slug        string
	Name        string
	ThemeURI    string
	Description string
	Author      string
	AuthorURI   string
	Version     string
	Template    string
	Status      string
	Tags        string
	TextDomain  string
	RequiresWP  string
	RequiresPHP string
}

var themeHeaderMap = []struct {
	field string
	label string
}{
	{"Name", "Theme Name"},
	{"ThemeURI", "Theme URI"},
	{"Description", "Description"},
	{"Author", "Author"},
	{"AuthorURI", "Author URI"},
	{"Version", "Version"},
	{"Template", "Template"},
	{"Status", "Status"},
	{"Tags", "Tags"},
	{"TextDomain", "Text Domain"},
	{"RequiresWP", "Requires at least"},
	{"RequiresPHP", "Requires PHP"},
}

// GetThemeData parses WordPress theme headers from style.css (first 8 KiB).
func GetThemeData(styleCSS string) (ThemeData, error) {
	f, err := os.Open(styleCSS)
	if err != nil {
		return ThemeData{}, err
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return ThemeData{}, err
	}
	headers := parseThemeHeaders(buf[:n])
	return ThemeData{
		Name:        headers["Name"],
		ThemeURI:    headers["ThemeURI"],
		Description: headers["Description"],
		Author:      headers["Author"],
		AuthorURI:   headers["AuthorURI"],
		Version:     headers["Version"],
		Template:    headers["Template"],
		Status:      headers["Status"],
		Tags:        headers["Tags"],
		TextDomain:  headers["TextDomain"],
		RequiresWP:  headers["RequiresWP"],
		RequiresPHP: headers["RequiresPHP"],
	}, nil
}

func parseThemeHeaders(raw []byte) map[string]string {
	out := make(map[string]string, len(themeHeaderMap))
	for _, h := range themeHeaderMap {
		out[h.field] = ""
	}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		for _, h := range themeHeaderMap {
			prefix := h.label + ":"
			if strings.HasPrefix(line, prefix) {
				out[h.field] = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
	}
	return out
}

// ThemesDir returns absPath/wp-content/themes.
func ThemesDir(absPath string) string {
	root := strings.TrimSpace(absPath)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "wp-content", "themes")
}

// IsBlockTheme reports whether the theme has templates/index.html (or legacy
// block-templates/index.html), matching WP_Theme::is_block_theme (P4.59).
func IsBlockTheme(absPath, stylesheet string) bool {
	stylesheet = strings.TrimSpace(stylesheet)
	if absPath == "" || stylesheet == "" {
		return false
	}
	dir := filepath.Join(ThemesDir(absPath), stylesheet)
	for _, rel := range []string{"templates/index.html", "block-templates/index.html"} {
		if st, err := os.Stat(filepath.Join(dir, rel)); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

// ListInstalledThemes scans wp-content/themes for style.css headers (P7.34).
func ListInstalledThemes(absPath string) ([]ThemeData, error) {
	dir := ThemesDir(absPath)
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
	var out []ThemeData
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		style := filepath.Join(dir, name, "style.css")
		data, err := GetThemeData(style)
		if err != nil || data.Name == "" {
			continue
		}
		data.Slug = name
		out = append(out, data)
	}
	return out, nil
}
