package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestInstallPlaceholderHomeAndInstallPHP(t *testing.T) {
	c := wp.NewContext(nil)
	c.Options = wp.NewOptions(nil, wp.NewObjectCache())
	srv := &server.Server{WP: c, Front: &server.Front{}}

	for _, path := range []string{"/", "/wp-admin/install.php"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "Installation") || !strings.Contains(body, "not installed") {
			t.Fatalf("%s unexpected body: %q", path, body)
		}
	}
}

func TestInstalledSiteHomeUnchanged(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "not installed yet") {
		t.Fatalf("installed site served install placeholder: %q", body)
	}
	if !strings.Contains(body, "GoPress Fixture") {
		t.Fatalf("expected normal home, got %q", body)
	}
}

func TestInstalledInstallPHPUnchanged(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/install.php", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	// Previously 404 (admin stub does not handle install.php); keep that.
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%q want 404", rec.Code, rec.Body.String())
	}
}
