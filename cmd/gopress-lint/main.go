// Command gopress-lint is a multichecker for GoPress custom analyzers.
// Analyzers are added in later tasks: sqlconcat (P1.3+), hooknames (P1.4b),
// nofmtprint (P1.5), trustedcomment (P3.4).
//
// sqlconcat (planned): flag string concatenation / fmt.Sprintf into SQL
// query strings; callers should use internal/db.Prepare (%s/%d/%f → ?)
// and never interpolate untrusted values into SQL.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	// Later tasks append analyzers: sqlconcat, hooknames, nofmtprint, trustedcomment.
	multichecker.Main(noopAnalyzer)
}
