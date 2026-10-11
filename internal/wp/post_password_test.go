package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetThePasswordForm(t *testing.T) {
	c := bootWP(t)
	html := wp.GetThePasswordForm(c, &wp.Post{ID: 42, Title: "Secret", Name: "secret", Type: "post"}, "http://127.0.0.1:8081/?p=42")
	for _, want := range []string{
		`class="post-password-form"`,
		`action=`,
		`action=postpass`,
		`name="post_password"`,
		`id="pwbox-42"`,
		`name="redirect_to"`,
		`value="http://127.0.0.1:8081/?p=42"`,
		`required`,
		`password-protected`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
}

func TestGetThePasswordFormInvalidPassword(t *testing.T) {
	c := bootWP(t)
	post := &wp.Post{ID: 42, Title: "Secret", Name: "secret", Type: "post", Password: "secret"}
	permalink := "http://127.0.0.1:8081/?p=42"
	req := httptest.NewRequest(http.MethodGet, "/?p=42", nil)
	req.Header.Set("Referer", permalink)
	req.AddCookie(&http.Cookie{Name: "wp-postpass_" + c.CookieHash(), Value: wp.HashPasswordPortable("wrong", 8)})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)

	html := wp.GetThePasswordForm(c, post, permalink)
	for _, want := range []string{
		`password-form-error`,
		`post-password-form-invalid-password`,
		`Invalid password.`,
		`aria-describedby="error-pwbox-42"`,
		`role="alert"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
}

func TestGetThePasswordFormFilter(t *testing.T) {
	c := bootWP(t)
	c.Hooks.AddFilter("the_password_form", func(value any, args ...any) any {
		return "CUSTOM_FORM"
	}, 10, 3)
	html := wp.GetThePasswordForm(c, &wp.Post{ID: 1, Type: "post"}, "")
	if html != "CUSTOM_FORM" {
		t.Fatalf("filter: %q", html)
	}
}

func TestPostPasswordRequired(t *testing.T) {
	c := bootWP(t)
	post := &wp.Post{ID: 1, Password: "secret", Title: "Hi"}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	if !wp.PostPasswordRequired(c, req, post) {
		t.Fatal("expected required without cookie")
	}
	hash := wp.HashPasswordPortable("secret", 8)
	req.AddCookie(&http.Cookie{Name: "wp-postpass_" + c.CookieHash(), Value: hash})
	if wp.PostPasswordRequired(c, req, post) {
		t.Fatal("expected unlocked with matching cookie")
	}
	req2 := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	req2.AddCookie(&http.Cookie{Name: "wp-postpass_" + c.CookieHash(), Value: wp.HashPasswordPortable("wrong", 8)})
	if !wp.PostPasswordRequired(c, req2, post) {
		t.Fatal("expected still required with wrong password cookie")
	}
	if wp.PostPasswordRequired(c, req, &wp.Post{ID: 2, Password: ""}) {
		t.Fatal("unprotected post must not require password")
	}
}
