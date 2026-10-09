// Command gopress-tools hosts development utilities (golden, capture, diff, …).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/robertbaehr/GoPress/internal/tools/depscheck"
	"github.com/robertbaehr/GoPress/internal/tools/diffharness"
	"github.com/robertbaehr/GoPress/internal/tools/porting"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "porting-status":
		if err := porting.WriteStatus("PORTING.md", "tools/porting-map.yaml", "php-reference"); err != nil {
			fmt.Fprintf(os.Stderr, "porting-status: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("wrote PORTING.md")
	case "diff":
		if err := runDiff(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "diff: %v\n", err)
			os.Exit(1)
		}
	case "capture":
		if err := runCapture(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "capture: %v\n", err)
			os.Exit(1)
		}
	case "deps-check":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "deps-check: %v\n", err)
			os.Exit(1)
		}
		if err := depscheck.Run(wd); err != nil {
			fmt.Fprintf(os.Stderr, "deps-check: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runDiff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	ref := fs.String("ref", "http://127.0.0.1:8081", "reference (PHP oracle) base URL")
	goURL := fs.String("go", "http://127.0.0.1:8081", "GoPress base URL (defaults to oracle until GoPress serves)")
	corpus := fs.String("corpus", "tests/corpus", "corpus directory")
	allow := fs.String("allowlist", "tests/diff/allowlist.yaml", "allowlist path")
	report := fs.String("report", "tests/diff/report.html", "HTML report output")
	skipDB := fs.Bool("skip-db", false, "skip reloading site.sql into gopress_ref/gopress_go")
	if err := fs.Parse(args); err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	return diffharness.RunDiff(diffharness.DiffOptions{
		RepoRoot:   wd,
		CorpusDir:  *corpus,
		AllowPath:  *allow,
		ReportPath: *report,
		RefURL:     *ref,
		GoURL:      *goURL,
		SkipDB:     *skipDB,
	})
}

func runCapture(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	base := fs.String("url", "http://127.0.0.1:8081", "target base URL")
	corpus := fs.String("corpus", "tests/corpus", "corpus directory")
	out := fs.String("out", "tests/diff/captures", "output directory")
	skipDB := fs.Bool("skip-db", false, "skip reloading site.sql")
	if err := fs.Parse(args); err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	return diffharness.RunCapture(diffharness.CaptureOptions{
		RepoRoot:  wd,
		CorpusDir: *corpus,
		OutDir:    *out,
		BaseURL:   *base,
		SkipDB:    *skipDB,
	})
}

func usage() {
	fmt.Fprintf(os.Stderr, `gopress-tools — GoPress development utilities

Usage:
  gopress-tools porting-status
  gopress-tools diff [--ref URL] [--go URL] [--corpus DIR] [--report PATH]
  gopress-tools capture [--url URL] [--corpus DIR] [--out DIR]
  gopress-tools deps-check

P0.7: diff compares the PHP oracle against GoPress (both may point at the oracle).
P0.8: deps-check enforces deps/allowlist.yaml and third_party LICENSE.txt.
`)
}
