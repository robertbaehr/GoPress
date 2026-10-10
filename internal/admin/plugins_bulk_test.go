package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestPluginsBulkActivateDeactivate(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")
	})
	_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")

	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/plugins.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="bulk-action-form"`,
		`id="bulk-action-selector-top"`,
		`activate-selected`,
		`deactivate-selected`,
		`name="checked[]"`,
		`value="hello.php"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in plugins list", want)
		}
	}
	nonce := extractHiddenNonce(body)
	if nonce == "" {
		t.Fatalf("missing bulk-plugins nonce")
	}

	form := url.Values{}
	form.Set("action", "activate-selected")
	form.Set("_wpnonce", nonce)
	form.Add("checked[]", "hello.php")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/plugins.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("bulk activate status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "activate=true") || !strings.Contains(loc, "bulk=1") {
		t.Fatalf("Location=%q", loc)
	}
	if !plugins.IsPluginActive(c, "hello.php") {
		t.Fatal("expected hello.php active after bulk activate")
	}

	req = httptest.NewRequest(http.MethodGet, loc, nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body = rec.Body.String()
	if !strings.Contains(body, "Selected plugins") || !strings.Contains(body, "activated") {
		t.Fatalf("expected bulk activate notice: %q", body)
	}
	nonce = extractHiddenNonce(body)

	form = url.Values{}
	form.Set("action", "deactivate-selected")
	form.Set("_wpnonce", nonce)
	form.Add("checked[]", "hello.php")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/plugins.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("bulk deactivate status=%d", rec.Code)
	}
	if plugins.IsPluginActive(c, "hello.php") {
		t.Fatal("expected hello.php inactive after bulk deactivate")
	}
}

var reHiddenNonce = regexp.MustCompile(`name="_wpnonce" value="([^"]+)"`)

func extractHiddenNonce(body string) string {
	m := reHiddenNonce.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
