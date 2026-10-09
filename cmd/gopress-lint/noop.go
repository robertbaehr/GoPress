package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// noopAnalyzer is a placeholder until real analyzers land (P1.3+).
var noopAnalyzer = &analysis.Analyzer{
	Name: "gopressnoop",
	Doc:  "placeholder analyzer; always succeeds",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, f := range pass.Files {
			_ = ast.FileExports(f)
		}
		return nil, nil
	},
}
