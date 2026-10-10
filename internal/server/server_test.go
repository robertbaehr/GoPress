package server_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestStaticAssetServed(t *testing.T) {
	root := t.TempDir()
	cssDir := filepath.Join(root, "wp-includes", "css")
	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := "body{color:red}"
	if err := os.WriteFile(filepath.Join(cssDir, "dashicons.min.css"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &server.Server{AbsPath: root}
	req := httptest.NewRequest(http.MethodGet, "/wp-includes/css/dashicons.min.css", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body != want {
		t.Fatalf("body=%q want %q", body, want)
	}
}

func TestStaticPathTraversalRejected(t *testing.T) {
	root := t.TempDir()
	secret := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	cssDir := filepath.Join(root, "wp-includes", "css")
	if err := os.MkdirAll(cssDir, 0o755); err != nil {
		t.Fatal(err)
	}

	s := &server.Server{AbsPath: root}
	req := httptest.NewRequest(http.MethodGet, "/wp-includes/css/../../secret.txt", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "nope") {
		t.Fatalf("path traversal served secret: status=%d", rec.Code)
	}
}

func TestWPLoginStub(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestWPJSONStub(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "json") {
		t.Fatalf("content-type=%q", ct)
	}
}

type stubFront struct{ called bool }

func (f *stubFront) ServeFront(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	f.called = true
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "front")
}

func TestFrontController(t *testing.T) {
	front := &stubFront{}
	s := &server.Server{Front: front, WP: wp.NewContext(context.Background())}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if !front.called {
		t.Fatal("front not called")
	}
	if rec.Code != http.StatusOK || rec.Body.String() != "front" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestFrontNilReturns503(t *testing.T) {
	s := &server.Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}
