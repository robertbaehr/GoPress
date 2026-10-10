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

	"github.com/robertbaehr/GoPress/internal/config"
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

func TestWPJSONIndex(t *testing.T) {
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	s := &server.Server{WP: c, AbsPath: cfg.ABSPATH}
	req := httptest.NewRequest(http.MethodGet, "/wp-json/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "json") {
		t.Fatalf("content-type=%q", ct)
	}
	if !strings.Contains(rec.Body.String(), `"namespaces"`) {
		t.Fatalf("expected namespaces in body: %s", rec.Body.String())
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
