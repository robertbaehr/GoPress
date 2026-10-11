package login_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.58: login_init then login_form_login fire before login_enqueue_scripts/head.
func TestLoginInitAndFormActionP658(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	var order []string
	c.Hooks.AddAction("login_init", func(_ ...any) {
		order = append(order, "init")
	}, 10, 0)
	c.Hooks.AddAction("login_form_login", func(_ ...any) {
		order = append(order, "form_login")
	}, 10, 0)
	c.Hooks.AddAction("login_enqueue_scripts", func(_ ...any) {
		order = append(order, "enqueue")
	}, 10, 0)
	c.Hooks.AddAction("login_head", func(_ ...any) {
		order = append(order, "head")
	}, 10, 0)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	want := []string{"init", "form_login", "enqueue", "head"}
	if len(order) != len(want) {
		t.Fatalf("order=%v want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v want %v", order, want)
		}
	}
}

// P6.58: lostpassword fires login_form_lostpassword after login_init.
func TestLoginInitLostPasswordFormActionP658(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	var order []string
	c.Hooks.AddAction("login_init", func(_ ...any) {
		order = append(order, "init")
	}, 10, 0)
	c.Hooks.AddAction("login_form_lostpassword", func(_ ...any) {
		order = append(order, "form_lp")
	}, 10, 0)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=lostpassword", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	want := []string{"init", "form_lp"}
	if len(order) != len(want) {
		t.Fatalf("order=%v want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v want %v", order, want)
		}
	}
}
