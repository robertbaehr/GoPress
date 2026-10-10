package admin_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

var heartbeatNonceRe = regexp.MustCompile(`heartbeatSettings\s*=\s*\{[^}]*"nonce"\s*:\s*"([^"]+)"`)

func extractHeartbeatNonce(html string) string {
	m := heartbeatNonceRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

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
	req0 := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req0.AddCookie(ck)
	rec0 := httptest.NewRecorder()
	srv.ServeHTTP(rec0, req0)
	nonce := extractHeartbeatNonce(rec0.Body.String())
	if nonce == "" {
		t.Fatal("missing heartbeatSettings nonce in admin HTML")
	}
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/admin-ajax.php", strings.NewReader("action=heartbeat&_nonce="+nonce+"&screen_id=dashboard"))
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
	if resp["nonces_expired"] != nil {
		t.Fatalf("unexpected nonces_expired: %#v", resp)
	}
}

func TestAdminAjaxHeartbeatMissingNonce(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	ck := loginCookie(t, c)
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/admin-ajax.php", strings.NewReader("action=heartbeat&screen_id=dashboard"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"success":false`) {
		t.Fatalf("expected json error, got %q", rec.Body.String())
	}
}

func TestAdminAjaxHeartbeatExpiredNonce(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	ck := loginCookie(t, c)
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/admin-ajax.php", strings.NewReader("action=heartbeat&_nonce=deadbeef01&screen_id=dashboard"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp["nonces_expired"] != true {
		t.Fatalf("expected nonces_expired, got %#v", resp)
	}
	if resp["heartbeat-nonce"] == nil || resp["rest-nonce"] == nil {
		t.Fatalf("expected refreshed nonces, got %#v", resp)
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
