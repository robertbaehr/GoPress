package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func bodyClassAttr(body string) string {
	i := strings.Index(body, `<body class="`)
	if i < 0 {
		return ""
	}
	rest := body[i:]
	j := strings.Index(rest, `">`)
	if j < 0 {
		return rest
	}
	return rest[:j+2]
}

// P7.137: admin menu includes collapse control and auto-fold body class.
func TestAdminMenuCollapseChrome(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="collapse-menu"`) {
		t.Fatalf("missing collapse-menu")
	}
	if !strings.Contains(body, `id="collapse-button"`) {
		t.Fatalf("missing collapse-button")
	}
	bc := bodyClassAttr(body)
	if !strings.Contains(bc, "auto-fold") {
		t.Fatalf("expected auto-fold in %s", bc)
	}
	if strings.Contains(bc, "folded") {
		t.Fatalf("unexpected folded without mfold: %s", bc)
	}
}

// P7.137: mfold=f cookie adds folded before auto-fold.
func TestAdminMenuFoldedFromSettingsCookie(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	req.AddCookie(&http.Cookie{Name: "wp-settings-1", Value: "mfold=f"})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	bc := bodyClassAttr(rec.Body.String())
	foldedIdx := strings.Index(bc, "folded")
	autoIdx := strings.Index(bc, "auto-fold")
	if foldedIdx < 0 || autoIdx < 0 {
		t.Fatalf("expected folded and auto-fold in %s", bc)
	}
	if foldedIdx > autoIdx {
		t.Fatalf("folded should precede auto-fold: %s", bc)
	}
}
