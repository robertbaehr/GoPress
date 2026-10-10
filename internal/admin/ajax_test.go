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
	if resp["wp-auth-check"] != false {
		t.Fatalf("expected wp-auth-check:false, got %#v", resp)
	}
	if _, ok := resp["server_time"]; !ok {
		t.Fatalf("expected server_time, got %#v", resp)
	}
	if _, ok := resp["success"]; ok {
		t.Fatalf("unexpected success field: %#v", resp)
	}
}

func TestAdminAjaxHeartbeatAuth(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	ck := loginCookie(t, c)
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/admin-ajax.php", strings.NewReader("action=heartbeat&_nonce=x&screen_id=dashboard"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%q", err, rec.Body.String())
	}
	if resp["wp-auth-check"] != true {
		t.Fatalf("expected wp-auth-check:true, got %#v", resp)
	}
}

func TestAdminAjaxRestNonceContentType(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	ck := loginCookie(t, c)
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/admin-ajax.php?action=rest-nonce", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type=%q want text/html", ct)
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

func TestAdminAjaxAuthCheck(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/admin-ajax.php?action=wp-auth-check", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("anon status=%d body=%q", rec.Code, rec.Body.String())
	}
	var anon map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &anon); err != nil {
		t.Fatalf("anon json: %v", err)
	}
	if anon["wp-auth-check"] != false {
		t.Fatalf("anon expected false, got %#v", anon)
	}

	ck := loginCookie(t, c)
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/admin-ajax.php", strings.NewReader("action=wp-auth-check"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth status=%d body=%q", rec.Code, rec.Body.String())
	}
	var auth map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &auth); err != nil {
		t.Fatalf("auth json: %v", err)
	}
	if auth["wp-auth-check"] != true {
		t.Fatalf("auth expected true, got %#v", auth)
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
