package login_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestLoginForceSSLAdminRedirect(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	off := false
	on := true
	_ = wp.ForceSSLAdmin(&on)
	t.Cleanup(func() { _ = wp.ForceSSLAdmin(&off) })

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8082/wp-login.php", nil)
	req.Host = "127.0.0.1:8082"
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc != "https://127.0.0.1:8082/wp-login.php" {
		t.Fatalf("Location=%q", loc)
	}
}
