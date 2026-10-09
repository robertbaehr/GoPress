package diffharness

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CaseResult is one corpus comparison outcome.
type CaseResult struct {
	ID        string
	Path      string
	Role      string
	Compare   string
	Match     bool
	Allowed   bool
	Reason    string
	Summary   string
	RefStatus int
	GoStatus  int
}

// Report aggregates harness results.
type Report struct {
	RefURL   string
	GoURL    string
	Started  time.Time
	Finished time.Time
	Cases    []CaseResult
}

// MatchPercent returns matched (including allowlisted) / total * 100.
func (r Report) MatchPercent() float64 {
	if len(r.Cases) == 0 {
		return 100
	}
	ok := 0
	for _, c := range r.Cases {
		if c.Match || c.Allowed {
			ok++
		}
	}
	return 100 * float64(ok) / float64(len(r.Cases))
}

// StrictMatchPercent excludes allowlisted mismatches.
func (r Report) StrictMatchPercent() float64 {
	if len(r.Cases) == 0 {
		return 100
	}
	ok := 0
	for _, c := range r.Cases {
		if c.Match {
			ok++
		}
	}
	return 100 * float64(ok) / float64(len(r.Cases))
}

func (r Report) matched() int {
	n := 0
	for _, c := range r.Cases {
		if c.Match {
			n++
		}
	}
	return n
}

func (r Report) allowed() int {
	n := 0
	for _, c := range r.Cases {
		if c.Allowed {
			n++
		}
	}
	return n
}

func (r Report) failed() int {
	n := 0
	for _, c := range r.Cases {
		if !c.Match && !c.Allowed {
			n++
		}
	}
	return n
}

// WriteHTML writes tests/diff/report.html.
func WriteHTML(path string, r Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\"><head><meta charset=\"utf-8\">")
	b.WriteString("<title>GoPress diff report</title>")
	b.WriteString(`<style>
body{font-family:ui-sans-serif,system-ui,sans-serif;margin:2rem;background:#f6f7f8;color:#1d2327}
h1{font-size:1.4rem} table{border-collapse:collapse;width:100%;background:#fff}
th,td{border:1px solid #dcdcde;padding:.45rem .6rem;text-align:left;font-size:14px}
th{background:#f0f0f1}.ok{color:#007017}.fail{color:#b32d2e}.allow{color:#996800}
.meta{margin-bottom:1rem;line-height:1.5}
code{font-size:12px}
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>GoPress differential report</h1>")
	fmt.Fprintf(&b, `<div class="meta">
<div><strong>Ref:</strong> %s</div>
<div><strong>Go:</strong> %s</div>
<div><strong>When:</strong> %s → %s</div>
<div><strong>Score:</strong> %.1f%% match (%d/%d strict; %d allowlisted; %d failed)</div>
</div>`,
		html.EscapeString(r.RefURL),
		html.EscapeString(r.GoURL),
		r.Started.Format(time.RFC3339),
		r.Finished.Format(time.RFC3339),
		r.MatchPercent(),
		r.matched(), len(r.Cases), r.allowed(), r.failed(),
	)
	b.WriteString("<table><thead><tr><th>Status</th><th>ID</th><th>Role</th><th>Path</th><th>Compare</th><th>HTTP</th><th>Detail</th></tr></thead><tbody>")
	for _, c := range r.Cases {
		status, class := "MATCH", "ok"
		if c.Allowed {
			status, class = "ALLOW", "allow"
		} else if !c.Match {
			status, class = "FAIL", "fail"
		}
		fmt.Fprintf(&b, `<tr class="%s"><td>%s</td><td><code>%s</code></td><td>%s</td><td><code>%s</code></td><td>%s</td><td>%d / %d</td><td>%s</td></tr>`,
			class, status,
			html.EscapeString(c.ID),
			html.EscapeString(c.Role),
			html.EscapeString(c.Path),
			html.EscapeString(c.Compare),
			c.RefStatus, c.GoStatus,
			html.EscapeString(c.Summary),
		)
	}
	b.WriteString("</tbody></table></body></html>\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
