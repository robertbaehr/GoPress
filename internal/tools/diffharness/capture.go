package diffharness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CaptureOptions configures response capture.
type CaptureOptions struct {
	RepoRoot  string
	CorpusDir string
	OutDir    string
	BaseURL   string
	SkipDB    bool
}

// RunCapture replays the corpus against one target and writes raw responses.
func RunCapture(opts CaptureOptions) error {
	root := opts.RepoRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		root = wd
	}
	if opts.CorpusDir == "" {
		opts.CorpusDir = filepath.Join(root, "tests/corpus")
	}
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join(root, "tests/diff/captures")
	}
	if opts.BaseURL == "" {
		opts.BaseURL = defaultOracleURL
	}

	fmt.Fprintf(os.Stderr, "capture: preparing fixture\n")
	if err := PrepareFixture(SetupOptions{RepoRoot: root, SkipDB: opts.SkipDB}); err != nil {
		return err
	}
	corpus, err := LoadCorpus(opts.CorpusDir)
	if err != nil {
		return err
	}
	target, err := newTarget("capture", opts.BaseURL)
	if err != nil {
		return err
	}
	roles := map[string]struct{}{}
	for _, e := range corpus {
		roles[e.Role] = struct{}{}
	}
	for role := range roles {
		if err := target.ensureRole(role); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}
	for _, e := range corpus {
		resp, err := target.do(e)
		if err != nil {
			return fmt.Errorf("%s: %w", e.ID, err)
		}
		meta := map[string]any{
			"id":      e.ID,
			"role":    e.Role,
			"method":  e.Method,
			"path":    e.Path,
			"status":  resp.Status,
			"headers": resp.Headers,
		}
		metaPath := filepath.Join(opts.OutDir, e.ID+".meta.json")
		bodyPath := filepath.Join(opts.OutDir, e.ID+".body")
		mb, _ := json.MarshalIndent(meta, "", "  ")
		if err := os.WriteFile(metaPath, mb, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(bodyPath, resp.Body, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "capture: %s → %s (%d bytes, status %d)\n", e.ID, bodyPath, len(resp.Body), resp.Status)
	}
	fmt.Printf("captured %d responses into %s\n", len(corpus), opts.OutDir)
	return nil
}
