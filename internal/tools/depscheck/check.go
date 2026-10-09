// Package depscheck implements gopress-tools deps-check (P0.8).
package depscheck

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type allowlist struct {
	Modules                    []modEntry `yaml:"modules"`
	Indirect                   []modEntry `yaml:"indirect"`
	ForbiddenLicenseSubstrings []string   `yaml:"forbidden_license_substrings"`
}

type modEntry struct {
	Path    string `yaml:"path"`
	License string `yaml:"license"`
	Purpose string `yaml:"purpose"`
}

// Run checks go.mod against deps/allowlist.yaml and third_party LICENSE.txt files.
func Run(repoRoot string) error {
	alPath := filepath.Join(repoRoot, "deps", "allowlist.yaml")
	b, err := os.ReadFile(alPath)
	if err != nil {
		return err
	}
	var al allowlist
	if err := yaml.Unmarshal(b, &al); err != nil {
		return fmt.Errorf("parse allowlist: %w", err)
	}
	allowed := map[string]modEntry{}
	for _, m := range al.Modules {
		allowed[m.Path] = m
	}
	for _, m := range al.Indirect {
		allowed[m.Path] = m
	}

	direct, err := parseGoModRequires(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return err
	}

	var errs []string
	for _, path := range direct {
		entry, ok := allowed[path]
		if !ok {
			errs = append(errs, fmt.Sprintf("unlisted direct dependency: %s", path))
			continue
		}
		if licenseForbidden(entry.License, al.ForbiddenLicenseSubstrings) {
			errs = append(errs, fmt.Sprintf("forbidden license on %s: %s", path, entry.License))
		}
	}

	// Every allowlisted direct module must still be in go.mod (stale allowlist).
	directSet := map[string]bool{}
	for _, p := range direct {
		directSet[p] = true
	}
	for _, m := range al.Modules {
		if !directSet[m.Path] {
			errs = append(errs, fmt.Sprintf("allowlisted module missing from go.mod: %s", m.Path))
		}
		if licenseForbidden(m.License, al.ForbiddenLicenseSubstrings) {
			errs = append(errs, fmt.Sprintf("forbidden license on allowlisted %s: %s", m.Path, m.License))
		}
	}

	third := filepath.Join(repoRoot, "third_party")
	if st, err := os.Stat(third); err == nil && st.IsDir() {
		entries, _ := os.ReadDir(third)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			lic := filepath.Join(third, e.Name(), "LICENSE.txt")
			if _, err := os.Stat(lic); err != nil {
				errs = append(errs, fmt.Sprintf("third_party/%s missing LICENSE.txt", e.Name()))
			}
		}
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "deps-check:", e)
		}
		return fmt.Errorf("%d dependency policy violation(s)", len(errs))
	}
	fmt.Println("deps-check: ok")
	return nil
}

func licenseForbidden(lic string, needles []string) bool {
	u := strings.ToUpper(lic)
	for _, n := range needles {
		n = strings.ToUpper(n)
		if n == "GPL" {
			// Match GPL/AGPL but not "NOT GPL" — treat AGPL separately via AGPL needle.
			if strings.Contains(u, "AGPL") {
				continue
			}
			if strings.Contains(u, "GPL") {
				return true
			}
			continue
		}
		if strings.Contains(u, n) {
			return true
		}
	}
	return false
}

func parseGoModRequires(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	inBlock := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "require (") {
			inBlock = true
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if line == "" || strings.HasPrefix(line, "//") {
				continue
			}
			if strings.Contains(line, "// indirect") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				out = append(out, fields[0])
			}
			continue
		}
		if strings.HasPrefix(line, "require ") && !strings.HasPrefix(line, "require (") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && !strings.Contains(line, "// indirect") {
				out = append(out, fields[1])
			}
		}
	}
	return out, sc.Err()
}
