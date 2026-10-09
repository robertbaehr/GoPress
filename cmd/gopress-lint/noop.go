package main

import (
	"golang.org/x/tools/go/analysis"
)

// noopAnalyzer is a placeholder until real analyzers land (P1.3+).
var noopAnalyzer = &analysis.Analyzer{
	Name: "gopressnoop",
	Doc:  "placeholder analyzer; always succeeds",
	Run: func(pass *analysis.Pass) (any, error) {
		_ = pass
		return nil, nil
	},
}
