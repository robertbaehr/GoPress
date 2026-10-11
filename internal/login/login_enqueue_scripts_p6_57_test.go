package login_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.57: login_enqueue_scripts then login_head fire before login_header.
func TestLoginEnqueueScriptsAndHeadP657(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	var order []string
	c.Hooks.AddAction("login_enqueue_scripts", func(_ ...any) {
		order = append(order, "enqueue")
	}, 10, 0)
	c.Hooks.AddAction("login_head", func(_ ...any) {
		order = append(order, "head")
	}, 10, 0)
	c.Hooks.AddAction("login_header", func(_ ...any) {
		order = append(order, "header")
	}, 10, 0)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	want := []string{"enqueue", "head", "header"}
	if len(order) != len(want) {
		t.Fatalf("order=%v want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v want %v", order, want)
		}
	}
}
