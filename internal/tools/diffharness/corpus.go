package diffharness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Entry is one corpus request.
type Entry struct {
	ID      string            `yaml:"id"`
	Role    string            `yaml:"role"`
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
	Body    *string           `yaml:"body"`
	Compare string            `yaml:"compare"`
	Nonce   *string           `yaml:"nonce"`
}

// LoadCorpus reads all *.yaml files under dir.
func LoadCorpus(dir string) ([]Entry, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no corpus yaml under %s", dir)
	}
	var all []Entry
	for _, m := range matches {
		raw, err := os.ReadFile(m)
		if err != nil {
			return nil, err
		}
		var entries []Entry
		if err := yaml.Unmarshal(raw, &entries); err != nil {
			return nil, fmt.Errorf("%s: %w", m, err)
		}
		for i := range entries {
			if entries[i].Method == "" {
				entries[i].Method = "GET"
			}
			if entries[i].Role == "" {
				entries[i].Role = "anon"
			}
			if entries[i].Compare == "" {
				entries[i].Compare = "raw"
			}
			if entries[i].Headers == nil {
				entries[i].Headers = map[string]string{}
			}
			if entries[i].ID == "" {
				return nil, fmt.Errorf("%s: entry missing id", m)
			}
			if !strings.HasPrefix(entries[i].Path, "/") {
				return nil, fmt.Errorf("%s: %s path must start with /", m, entries[i].ID)
			}
		}
		all = append(all, entries...)
	}
	return all, nil
}
