package login_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.69: lostpassword_form fires on the lost-password screen.
func TestLostPasswordFormActionP669(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	var fired int
	c.Hooks.AddAction("lostpassword_form", func(args ...any) {
		fired++
	}, 10, 0)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=lostpassword", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if fired != 1 {
		t.Fatalf("fired=%d", fired)
	}
}
