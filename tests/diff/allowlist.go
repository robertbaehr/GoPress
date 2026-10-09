package diff

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AllowEntry exempts a corpus case from mismatch failures until remove_by.
type AllowEntry struct {
	ID       string `yaml:"id"`
	Path     string `yaml:"path"`
	Reason   string `yaml:"reason"`
	RemoveBy string `yaml:"remove_by"`
}

// LoadAllowlist reads allowlist.yaml (empty list or sequence of entries).
func LoadAllowlist(path string) ([]AllowEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw = bytesTrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}
	var entries []AllowEntry
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("allowlist: %w", err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	var active []AllowEntry
	for _, e := range entries {
		if e.RemoveBy != "" {
			deadline, err := time.Parse("2006-01-02", e.RemoveBy)
			if err != nil {
				return nil, fmt.Errorf("allowlist %q: bad remove_by %q", e.ID, e.RemoveBy)
			}
			if today.After(deadline) {
				continue
			}
		}
		active = append(active, e)
	}
	return active, nil
}

// Allowed reports whether id/path is covered by the allowlist.
func Allowed(entries []AllowEntry, id, path string) (AllowEntry, bool) {
	for _, e := range entries {
		if e.ID != "" && e.ID == id {
			return e, true
		}
		if e.Path != "" && (e.Path == path || strings.HasPrefix(path, e.Path)) {
			return e, true
		}
	}
	return AllowEntry{}, false
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
