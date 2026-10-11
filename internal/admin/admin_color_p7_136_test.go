package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.136: admin chrome uses the user's admin_color usermeta (editor=coffee).
func TestAdminColorSchemeCoffeeEditor(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "editor")
	if err != nil || user == nil {
		t.Fatalf("editor: %v", err)
	}
	if got := wp.GetUserOption(c, "admin_color", user.ID); got != "coffee" {
		t.Fatalf("fixture admin_color=%q want coffee", got)
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
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()[:min(200, rec.Body.Len())])
	}
	body := rec.Body.String()
	if !strings.Contains(body, `admin-color-coffee`) {
		t.Fatalf("expected body class admin-color-coffee")
	}
	if !strings.Contains(body, `/wp-admin/css/colors/coffee/colors.min.css`) {
		t.Fatalf("expected coffee colors.css")
	}
	if strings.Contains(body, `admin-color-modern`) {
		t.Fatalf("editor should not use modern")
	}
}
