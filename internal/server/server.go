// Package server is the GoPress HTTP front controller (P3.1+).
package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// FrontHandler handles non-static front-end requests after rewrite routing.
type FrontHandler interface {
	ServeFront(w http.ResponseWriter, r *http.Request, c *wp.Context)
}

// Server routes static assets, stubs, and the front controller.
type Server struct {
	// WP is the process-wide boot context (shared Options/DB/Hooks/…).
	WP *wp.Context

	// AbsPath is WordPress ABSPATH (trailing slash optional).
	AbsPath string

	// Front handles everything that is not static or a stub route.
	// When nil, those requests return 503.
	Front FrontHandler

	// REST serves /wp-json. When nil, a default rest.Server is used.
	REST *rest.Server
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		http.Error(w, "GoPress server unavailable\n", http.StatusServiceUnavailable)
		return
	}
	path := cleanRequestPath(r.URL.Path)

	if isStaticPath(path) {
		s.serveStatic(w, r, path)
		return
	}
	if path == "/wp-login.php" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, "GoPress: wp-login.php not implemented\n")
		return
	}
	if path == "/wp-json" || strings.HasPrefix(path, "/wp-json/") {
		c := s.requestContext(w, r)
		rs := s.REST
		if rs == nil {
			rs = &rest.Server{}
		}
		rs.Handle(w, r, c)
		return
	}

	c := s.requestContext(w, r)
	if s.Front == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "GoPress not implemented\n")
		return
	}
	s.Front.ServeFront(w, r, c)
}

func (s *Server) requestContext(w http.ResponseWriter, r *http.Request) *wp.Context {
	if s.WP == nil {
		c := wp.NewContext(r.Context())
		c.Req = r
		c.Res = w
		return c
	}
	return s.WP.ForRequest(w, r)
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request, path string) {
	root := s.AbsPath
	if root == "" && s.WP != nil && s.WP.Config != nil {
		root = s.WP.Config.ABSPATH
	}
	if root == "" {
		http.NotFound(w, r)
		return
	}
	root = filepath.Clean(root)
	rel := strings.TrimPrefix(path, "/")
	full := filepath.Join(root, filepath.FromSlash(rel))
	// Prevent path escape outside ABSPATH.
	if !pathUnderRoot(full, root) {
		http.NotFound(w, r)
		return
	}
	st, err := os.Stat(full)
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, full)
}

func pathUnderRoot(full, root string) bool {
	full = filepath.Clean(full)
	root = filepath.Clean(root)
	sep := string(os.PathSeparator)
	if root != string(os.PathSeparator) && !strings.HasSuffix(root, sep) {
		root += sep
	}
	return full == strings.TrimSuffix(root, sep) || strings.HasPrefix(full, root)
}

func cleanRequestPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// Keep URL path style (forward slashes); collapse // and resolve .. via path.Clean semantics.
	cleaned := pathCleanURL(p)
	if cleaned == "" {
		return "/"
	}
	return cleaned
}

func pathCleanURL(p string) string {
	if p == "" {
		return "/"
	}
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return "/"
	}
	return "/" + strings.Join(out, "/")
}

// isStaticPath reports whether path is a frozen WordPress asset prefix.
func isStaticPath(path string) bool {
	prefixes := []string{
		"/wp-admin/css/",
		"/wp-admin/js/",
		"/wp-admin/images/",
		"/wp-includes/css/",
		"/wp-includes/js/",
		"/wp-includes/images/",
		"/wp-includes/fonts/",
		"/wp-content/uploads/",
		"/wp-content/themes/",
		"/wp-content/plugins/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
