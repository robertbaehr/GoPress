package main

import (
	"golang.org/x/tools/go/analysis"
)

// noopAnalyzer is a placeholder until real analyzers land.
// Next up: sqlconcat (flag SQL string concatenation; see main.go note).
var noopAnalyzer = &analysis.Analyzer{
	Name: "gopressnoop",
	Doc:  "placeholder analyzer; always succeeds (sqlconcat not yet implemented)",
	Run: func(pass *analysis.Pass) (any, error) {
		_ = pass
		return nil, nil
	},
}
