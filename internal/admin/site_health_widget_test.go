package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestDashboardSiteHealthWidget(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="dashboard_site_health"`) {
		t.Fatalf("missing site health widget: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "Site Health Status") {
		t.Fatalf("missing title: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "health-check-widget") {
		t.Fatalf("missing health-check-widget: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, `/wp-admin/site-health.php`) {
		t.Fatalf("missing site-health link: %q", body[:min(600, len(body))])
	}
}
