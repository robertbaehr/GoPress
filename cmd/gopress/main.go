// Command gopress is the GoPress server binary.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

const version = "0.0.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version":
		fmt.Println(version)
	case "serve":
		serveCmd(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `gopress — WordPress backend in Go

Usage:
  gopress serve --addr :8080 --root .
  gopress version
`)
}

func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	root := fs.String("root", ".", "WordPress root (ABSPATH)")
	_ = fs.Parse(args)

	cfg, err := config.Load(config.LoadOptions{Root: *root})
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	wpCtx, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "boot: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = wpCtx.DB.Close() }()

	blogname, _ := wpCtx.Options.GetOption("blogname", "").(string)
	log.Printf("gopress boot ok root=%s db=%s blogname=%q", *root, cfg.DBName, blogname)

	srv := &server.Server{
		WP:      wpCtx,
		AbsPath: cfg.ABSPATH,
		Front:   &server.Front{},
	}

	fmt.Fprintf(os.Stderr, "gopress serve listening on %s (root=%s)\n", *addr, *root)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		fmt.Fprintf(os.Stderr, "serve error: %v\n", err)
		os.Exit(1)
	}
}
