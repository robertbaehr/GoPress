package diffharness

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	testdiff "github.com/robertbaehr/GoPress/tests/diff"
)

// DiffOptions configures the differential run.
type DiffOptions struct {
	RepoRoot   string
	CorpusDir  string
	AllowPath  string
	ReportPath string
	RefURL     string
	GoURL      string
	SkipDB     bool
}

// RunDiff prepares fixtures, replays the corpus against both targets, and writes the report.
func RunDiff(opts DiffOptions) error {
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
	if opts.AllowPath == "" {
		opts.AllowPath = filepath.Join(root, "tests/diff/allowlist.yaml")
	}
	if opts.ReportPath == "" {
		opts.ReportPath = filepath.Join(root, "tests/diff/report.html")
	}
	if opts.RefURL == "" {
		opts.RefURL = defaultOracleURL
	}
	if opts.GoURL == "" {
		opts.GoURL = defaultOracleURL
	}

	fmt.Fprintf(os.Stderr, "diff: preparing fixture DBs/uploads\n")
	if err := PrepareFixture(SetupOptions{RepoRoot: root, SkipDB: opts.SkipDB}); err != nil {
		return err
	}

	corpus, err := LoadCorpus(opts.CorpusDir)
	if err != nil {
		return err
	}
	allow, err := testdiff.LoadAllowlist(opts.AllowPath)
	if err != nil {
		return err
	}

	ref, err := newTarget("ref", opts.RefURL)
	if err != nil {
		return err
	}
	goT, err := newTarget("go", opts.GoURL)
	if err != nil {
		return err
	}

	// Login once per role used in the corpus.
	roles := map[string]struct{}{}
	for _, e := range corpus {
		roles[e.Role] = struct{}{}
	}
	for role := range roles {
		fmt.Fprintf(os.Stderr, "diff: login role %s on ref\n", role)
		if err := ref.ensureRole(role); err != nil {
			return fmt.Errorf("ref login %s: %w", role, err)
		}
		fmt.Fprintf(os.Stderr, "diff: login role %s on go\n", role)
		if err := goT.ensureRole(role); err != nil {
			return fmt.Errorf("go login %s: %w", role, err)
		}
	}

	rep := Report{
		RefURL:  opts.RefURL,
		GoURL:   opts.GoURL,
		Started: time.Now().UTC(),
	}

	for _, e := range corpus {
		fmt.Fprintf(os.Stderr, "diff: %s %s %s\n", e.ID, e.Method, e.Path)
		refResp, err := ref.do(e)
		if err != nil {
			return fmt.Errorf("%s ref: %w", e.ID, err)
		}
		goResp, err := goT.do(e)
		if err != nil {
			return fmt.Errorf("%s go: %w", e.ID, err)
		}
		nRef, errRef := testdiff.Normalize(refResp, e.Compare)
		nGo, errGo := testdiff.Normalize(goResp, e.Compare)
		cr := CaseResult{
			ID:        e.ID,
			Path:      e.Path,
			Role:      e.Role,
			Compare:   e.Compare,
			RefStatus: refResp.Status,
			GoStatus:  goResp.Status,
		}
		if errRef != nil || errGo != nil {
			// Still compare best-effort normalized forms.
			cr.Summary = fmt.Sprintf("normalize: ref=%v go=%v", errRef, errGo)
		}
		if testdiff.Equal(nRef, nGo) {
			cr.Match = true
			cr.Summary = "match"
		} else {
			cr.Match = false
			cr.Summary = testdiff.DiffSummary(nRef, nGo)
			if ae, ok := testdiff.Allowed(allow, e.ID, e.Path); ok {
				cr.Allowed = true
				cr.Reason = ae.Reason
				cr.Summary += " [allowlisted: " + ae.Reason + "]"
			}
		}
		rep.Cases = append(rep.Cases, cr)
	}
	rep.Finished = time.Now().UTC()

	if err := WriteHTML(opts.ReportPath, rep); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "diff: wrote %s\n", opts.ReportPath)
	fmt.Printf("match: %.1f%% (%d/%d strict matches, %d allowlisted, %d failed)\n",
		rep.MatchPercent(), rep.matched(), len(rep.Cases), rep.allowed(), rep.failed())

	if rep.failed() > 0 {
		return fmt.Errorf("differential match %.1f%% with %d failures", rep.MatchPercent(), rep.failed())
	}
	return nil
}
