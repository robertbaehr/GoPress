package admin_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestAdminAjaxHeartbeatAnon(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/admin-ajax.php?action=heartbeat", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%q", err, rec.Body.String())
	}
	if resp["success"] != true {
		t.Fatalf("expected success:true, got %#v", resp)
	}
}

func TestAdminAjaxRestNonceRequiresLogin(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/admin-ajax.php?action=rest-nonce", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || rec.Body.String() != "0" {
		t.Fatalf("anon: status=%d body=%q", rec.Code, rec.Body.String())
	}

	ck := loginCookie(t, c)
	req = httptest.NewRequest(http.MethodGet, "/wp-admin/admin-ajax.php?action=rest-nonce", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth status=%d body=%q", rec.Code, rec.Body.String())
	}
	nonce := strings.TrimSpace(rec.Body.String())
	if len(nonce) < 8 {
		t.Fatalf("expected nonce text, got %q", nonce)
	}
}

func TestAdminAjaxLoggedIn(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/admin-ajax.php?action=logged-in", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Body.String() != "0" {
		t.Fatalf("anon body=%q", rec.Body.String())
	}

	ck := loginCookie(t, c)
	req = httptest.NewRequest(http.MethodGet, "/wp-admin/admin-ajax.php?action=logged-in", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "1" {
		t.Fatalf("auth status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAdminAjaxEmptyAction(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/admin-ajax.php", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || rec.Body.String() != "0" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}
