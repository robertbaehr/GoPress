// Package theme — block pattern file loading from themes (P5.9).
package theme

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PatternFile is metadata + content extracted from a theme patterns/*.php file.
type PatternFile struct {
	Slug        string
	Title       string
	Description string
	Categories  []string
	BlockTypes  []string
	Inserter    bool
	Content     string
	FilePath    string
}

var (
	headerTitle       = regexp.MustCompile(`(?mi)^\s*\*\s*Title:\s*(.+?)\s*$`)
	headerSlug        = regexp.MustCompile(`(?mi)^\s*\*\s*Slug:\s*(.+?)\s*$`)
	headerDescription = regexp.MustCompile(`(?mi)^\s*\*\s*Description:\s*(.+?)\s*$`)
	headerCategories  = regexp.MustCompile(`(?mi)^\s*\*\s*Categories:\s*(.+?)\s*$`)
	headerBlockTypes  = regexp.MustCompile(`(?mi)^\s*\*\s*Block Types:\s*(.+?)\s*$`)
	headerInserter    = regexp.MustCompile(`(?mi)^\s*\*\s*Inserter:\s*(.+?)\s*$`)
	phpTagStrip       = regexp.MustCompile(`(?s)<\?(?:php|=)?.*?\?>`)
)

// LoadThemePatterns scans wp-content/themes/{stylesheet}/patterns/*.php.
func LoadThemePatterns(absPath, stylesheet string) ([]PatternFile, error) {
	stylesheet = strings.TrimSpace(stylesheet)
	if absPath == "" || stylesheet == "" {
		return nil, nil
	}
	dir := filepath.Join(absPath, "wp-content", "themes", stylesheet, "patterns")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]PatternFile, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".php") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		pf, ok := parsePatternFile(path, string(raw))
		if ok {
			out = append(out, pf)
		}
	}
	return out, nil
}

func parsePatternFile(path, raw string) (PatternFile, bool) {
	pf := PatternFile{FilePath: path, Inserter: true}
	if m := headerSlug.FindStringSubmatch(raw); len(m) > 1 {
		pf.Slug = strings.TrimSpace(m[1])
	}
	if pf.Slug == "" {
		return pf, false
	}
	if m := headerTitle.FindStringSubmatch(raw); len(m) > 1 {
		pf.Title = strings.TrimSpace(m[1])
	}
	if pf.Title == "" {
		pf.Title = pf.Slug
	}
	if m := headerDescription.FindStringSubmatch(raw); len(m) > 1 {
		pf.Description = strings.TrimSpace(m[1])
	}
	if m := headerCategories.FindStringSubmatch(raw); len(m) > 1 {
		pf.Categories = splitCSV(m[1])
	}
	if m := headerBlockTypes.FindStringSubmatch(raw); len(m) > 1 {
		pf.BlockTypes = splitCSV(m[1])
	}
	if m := headerInserter.FindStringSubmatch(raw); len(m) > 1 {
		v := strings.ToLower(strings.TrimSpace(m[1]))
		pf.Inserter = v != "no" && v != "false" && v != "0"
	}
	pf.Content = extractPatternContent(raw)
	return pf, true
}

func extractPatternContent(raw string) string {
	// Content is everything after the opening PHP docblock / close tag.
	idx := strings.Index(raw, "?>")
	body := raw
	if idx >= 0 {
		body = raw[idx+2:]
	}
	// Strip remaining inline PHP (theme patterns often echo URLs/strings).
	body = phpTagStrip.ReplaceAllString(body, "")
	return strings.TrimSpace(body)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
