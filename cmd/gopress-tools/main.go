// Command gopress-tools hosts development utilities (golden, capture, diff, …).
package main

import (
	"fmt"
	"os"

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
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `gopress-tools — GoPress development utilities

Usage:
  gopress-tools porting-status

Later (P0.7+): capture, diff, golden, fixture, hookgen, apigen, deps-check, build
`)
}
